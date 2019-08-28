package oex

// ErrCapture helps with error info
type ErrCapture struct {
	Error string `json:"code"`
	Msg   string `json:"msg"`
}

// TickerDataResponse returns the time and tickerinfo for a trading pair
type TickerDataResponse struct {
	High   float64 `json:"high,string"`
	Volume float64 `json:"vol,string"`
	Last   float64 `json:"last"`
	Low    float64 `json:"low,string"`
	Buy    float64 `json:"buy"`
	Sell   float64 `json:"sell"`
	Rose   string  `json:"rose"`
	Time   int64   `json:"time"`
}

// TickerResponse returns the time and tickerinfo for a trading pair
type TickerResponse struct {
	ErrCapture `json:",omitempty"`
	Data       TickerDataResponse `json:"data"`
}

// AllTicker stores the ticker price data for a currency pair
type AllTicker struct {
	Symbol string  `json:"symbol"`
	High   float64 `json:"high,string"`
	Volume float64 `json:"vol,string"`
	Last   float64 `json:"last"`
	Low    float64 `json:"low,string"`
	Buy    float64 `json:"buy"`
	Sell   float64 `json:"sell"`
	Rose   string  `json:"rose"`
}

// AllTickerDataResponse used for data storage
type AllTickerDataResponse struct {
	Date   int64       `json:"date"`
	Ticker []AllTicker `json:"ticker"`
}

// AllTickerResponse returns the time and Ticker info for all trading pairs
type AllTickerResponse struct {
	ErrCapture `json:",omitempty"`
	Data       AllTickerDataResponse `json:"data"`
}

// KlineResponse stores kline info for given currency exchange
type KlineResponse struct {
	ErrCapture `json:",omitempty"`
	Data       [][]float64 `json:"data"`
}

// TradeData returns market transaction records for a currency pair
type TradeData struct {
	Amount    float64 `json:"amount"`
	Price     float64 `json:"price,string"`
	ID        int64   `json:"id"`
	OrderType string  `json:"type"`
}

// TradeResponse returns market transaction records for a currency pair
type TradeResponse struct {
	ErrCapture `json:",omitempty"`
	Data       []TradeData `json:"data"`
}

// PairAccuracyResp stores accuracy for a currency pair
type PairAccuracyResp struct {
	Symbol          string `json:"symbol"`
	CountCoin       string `json:"count_coin"`
	AmountPrecision int64  `json:"amount_precision"`
	BaseCoin        string `json:"base_coin"`
	PricePrecision  int64  `json:"price_precision"`
}

// AllPairResponse stores accuracy for all currency pairs enabled on the exchange
type AllPairResponse struct {
	ErrCapture `json:",omitempty"`
	Data       []PairAccuracyResp `json:"data"`
}

// CoinInfo stores info about particular coins that user has
type CoinInfo struct {
	Coin        string  `json:"coin"`
	Normal      float64 `json:"normal,string"`
	Locked      float64 `json:"locked,string"`
	BtcValuatin string  `json:"btcValuatin"`
}

// UserInfoDataResponse stores user's balance info
type UserInfoDataResponse struct {
	Total    string     `json:"total_asset"`
	CoinData []CoinInfo `json:"coin_list"`
}

// UserInfoResponse stores user's balance info
type UserInfoResponse struct {
	ErrCapture `json:",omitempty"`
	Data       UserInfoDataResponse `json:"data"`
}

// OrderResponse stores order info
type OrderResponse struct {
	Side            string  `json:"side"`
	TotalPrice      float64 `json:"total_price"`
	CreateTime      int64   `json:"created_at"`
	AvgPrice        float64 `json:"avg_price"`
	CountCoin       string  `json:"countCoin"`
	Source          int64   `json:"source"`
	Type            int64   `json:"type"`
	SideMsg         string  `json:"side_msg"`
	Volume          float64 `json:"vol,string"`
	Price           float64 `json:"price,string"`
	SourceMsg       string  `json:"source_msg"`
	StatusMsg       string  `json:"status_msg"`
	DealVolume      float64 `json:"deal_volume,string"`
	OrderID         int64   `json:"id"`
	RemainingVolume string  `json:"remain_volume"`
	BaseCoin        string  `json:"baseCoin"`
	Status          int64   `json:"status"`
}

// AllOrderDataResponse stores info for multiple orders
type AllOrderDataResponse struct {
	Count     int64           `json:"count"`
	OrderList []OrderResponse `json:"orderList"`
}

// AllOrderResponse stores info for multiple orders
type AllOrderResponse struct {
	ErrCapture `json:",omitempty"`
	Data       AllOrderDataResponse `json:"data"`
}

// OrderHistoryData stores data about past orders
type OrderHistoryData struct {
	Volume    float64 `json:"vol,string"`
	Side      string  `json:"side"`
	FeeCoin   string  `json:"feeCoin"`
	Price     float64 `json:"price,string"`
	Fee       float64 `json:"fee,string"`
	CTime     int64   `json:"ctime"`
	DealPrice float64 `json:"deal_price,string"`
	ID        int64   `json:"id"`
	Type      string  `json:"type"`
	BidID     int64   `json:"bid_id"`
	AskID     int64   `json:"ask_id"`
	BidUserID int64   `json:"bid_user_id"`
	AskUserID int64   `json:"ask_user_id"`
}

// OrderDataResponse stores past orders
type OrderDataResponse struct {
	Count      int64              `json:"conut"`
	ResultList []OrderHistoryData `json:"resultList"`
}

// OrderHistoryResponse stores past orders
type OrderHistoryResponse struct {
	ErrCapture `json:",omitempty"`
	Data       OrderDataResponse `json:"data"`
}

// RemoveOrderResponse indicates whether the cancel order query was successful
type RemoveOrderResponse struct {
	ErrCapture `json:",omitempty"`
	Data       string `json:"data"`
}

// CreateOrderDataResponse stores orderID for successful orders creations
type CreateOrderDataResponse struct {
	OrderID int64 `json:"order_id"`
}

// CreateOrderResponse stores orderID for created orders
type CreateOrderResponse struct {
	ErrCapture `json:",omitempty"`
	Data       CreateOrderDataResponse `json:"data"`
}

// LatestCurrencyPrices stores latest price for currency pairs
type LatestCurrencyPrices struct {
	ErrCapture `json:",omitempty"`
	Data       map[string]float64 `json:"data"`
}

// MarketDepth stores market depth data
type MarketDepth struct {
	Asks [][]float64 `json:"asks"`
	Bids [][]float64 `json:"bids"`
}

// MarketDepthData stores market depth data
type MarketDepthData struct {
	Tick MarketDepth `json:"tick"`
}

// MarketDepthResponse stores market depth data
type MarketDepthResponse struct {
	ErrCapture `json:",omitempty"`
	Data       MarketDepthData `json:"data"`
}

// OpenOrderResponse stores data of all current delegation
type OpenOrderResponse struct {
	ErrCapture `json:",omitempty"`
	Data       OrderDataResponse `json:"data"`
}

// SelfTradeDataResponse stores self trade data
type SelfTradeDataResponse struct {
	OrderID int64 `json:"order_id"`
}

// SelfTradeResponse stores self trade data
type SelfTradeResponse struct {
	ErrCapture `json:",omitempty"`
	Data       SelfTradeDataResponse `json:"data"`
}

// BalanceInfo stores balance data
type BalanceInfo struct {
	Symbol  string  `json:"symbol"`
	Balance float64 `json:"balance"`
}

// DepositList stores deposit transfer data
type DepositList struct {
	UID       int64   `json:"uid"`
	Symbol    string  `json:"symbol"`
	Fee       float64 `json:"fee,string"`
	Amount    float64 `json:"amount"`
	CreatedAt string  `json:"created_at"`
}

// UserInfoData stores balance data
type UserInfoData struct {
	BalanceInfo []BalanceInfo `json:"balance_info"`
	DepositList []DepositList `json:"deposit_list"`
}

// UserAssetResponse stores user asset and recharge data
type UserAssetResponse struct {
	ErrCapture `json:",omitempty"`
	Data       UserInfoData `json:"data"`
}

// OrderData stores data for an order
type OrderData struct {
	ID         int64   `json:"id"`
	Side       string  `json:"side"`
	SideMsg    string  `json:"side_msg"`
	CreatedAt  string  `json:"created_at"`
	Price      float64 `json:"price,string"`
	Volume     float64 `json:"vol,string"`
	DealVolume float64 `json:"deal_volume,string"`
	TotalPrice float64 `json:"total_price"`
	Fee        float64 `json:"fee,string"`
	AvgPrice   float64 `json:"avg_price"`
}

// TradeData2 stores data for a trade
type TradeData2 struct {
	ID        int64   `json:"id"`
	CreatedAt string  `json:"created_at"`
	Price     float64 `json:"price,string"`
	Volume    float64 `json:"vol,string"`
	DealPrice float64 `json:"deal_price,string"`
	Fee       float64 `json:"fee,string"`
}

// FetchOrderDataResp stores data for a given orderid
type FetchOrderDataResp struct {
	OrderInfo OrderData    `json:"order_info"`
	TradeList []TradeData2 `json:"trade_list"`
}

// FetchOrderResponse stores data for a given order id
type FetchOrderResponse struct {
	ErrCapture `json:",omitempty"`
	Data       FetchOrderDataResp `json:"data"`
}
