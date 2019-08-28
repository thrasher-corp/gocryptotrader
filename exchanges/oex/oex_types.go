package oex

// ErrCapture helps with error info
type ErrCapture struct {
	Error string `json:"code"`
	Msg   string `json:"msg"`
}

// TickerTempResponse returns the time and tickerinfo for a trading pair
type TickerTempResponse struct {
	High   string  `json:"high"`
	Volume string  `json:"vol"`
	Last   float64 `json:"last"`
	Low    string  `json:"low"`
	Buy    float64 `json:"buy"`
	Sell   float64 `json:"sell"`
	Rose   string  `json:"rose"`
	Time   int64   `json:"time"`
}

// TickerResponse returns the time and tickerinfo for a trading pair
type TickerResponse struct {
	ErrCapture `json:",omitempty"`
	Data       TickerTempResponse `json:"data"`
}

// AllTicker stores the ticker price data for a currency pair
type AllTicker struct {
	Symbol string  `json:"symbol"`
	High   string  `json:"high"`
	Volume string  `json:"vol"`
	Last   float64 `json:"last"`
	Low    string  `json:"low"`
	Buy    float64 `json:"buy"`
	Sell   float64 `json:"sell"`
	Rose   string  `json:"rose"`
}

// AllTickerTempResponse used for temp storage
type AllTickerTempResponse struct {
	Date   int64       `json:"date"`
	Ticker []AllTicker `json:"ticker"`
}

// AllTickerResponse returns the time and Ticker info for all trading pairs
type AllTickerResponse struct {
	ErrCapture `json:",omitempty"`
	Data       AllTickerTempResponse `json:"data"`
}

// KlineResponse stores kline info for given currency exchange
type KlineResponse struct {
	ErrCapture `json:",omitempty"`
	Data       [][]float64 `json:"data"`
}

// TradeTemp returns market transaction records for a currency pair
type TradeTemp struct {
	Amount    float64 `json:"amount"`
	Price     float64 `json:"price"`
	ID        int64   `json:"id"`
	OrderType string  `json:"type"`
}

// TradeResponse returns market transaction records for a currency pair
type TradeResponse struct {
	ErrCapture `json:",omitempty"`
	Data       []TradeTemp `json:"data"`
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
	Coin        string `json:"coin"`
	Normal      string `json:"normal"`
	Locked      string `json:"locked"`
	BtcValuatin string `json:"btcValuatin"`
}

// UserInfoTempResponse stores user's balance info
type UserInfoTempResponse struct {
	Total    string     `json:"total_asset"`
	CoinData []CoinInfo `json:"coin_list"`
}

// UserInfoResponse stores user's balance info
type UserInfoResponse struct {
	ErrCapture `json:",omitempty"`
	Data       UserInfoTempResponse `json:"data"`
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
	Volume          string  `json:"volume"`
	Price           float64 `json:"price"`
	SourceMsg       string  `json:"source_msg"`
	StatusMsg       string  `json:"status_msg"`
	DealVolume      float64 `json:"deal_volume"`
	OrderID         int64   `json:"id"`
	RemainingVolume string  `json:"remain_volume"`
	BaseCoin        string  `json:"baseCoin"`
	Status          int64   `json:"status"`
}

// AllOrderTempResponse stores info for multiple orders
type AllOrderTempResponse struct {
	Count     int64           `json:"count"`
	OrderList []OrderResponse `json:"orderList"`
}

// AllOrderResponse stores info for multiple orders
type AllOrderResponse struct {
	ErrCapture `json:",omitempty"`
	Data       AllOrderTempResponse `json:"data"`
}

// OrderHistoryData stores data about past orders
type OrderHistoryData struct {
	Volume    float64 `json:"vol"`
	Side      string  `json:"side"`
	FeeCoin   string  `json:"feeCoin"`
	Price     string  `json:"price"`
	Fee       string  `json:"fee"`
	CTime     int64   `json:"ctime"`
	DealPrice string  `json:"deal_price"`
	ID        int64   `json:"id"`
	Type      string  `json:"type"`
	BidID     int64   `json:"bid_id"`
	AskID     int64   `json:"ask_id"`
	BidUserID int64   `json:"bid_user_id"`
	AskUserID int64   `json:"ask_user_id"`
}

// OrderTempResponse stores past orders
type OrderTempResponse struct {
	Count      int64              `json:"conut"`
	ResultList []OrderHistoryData `json:"resultList"`
}

// OrderHistoryResponse stores past orders
type OrderHistoryResponse struct {
	ErrCapture `json:",omitempty"`
	Data       OrderTempResponse `json:"data"`
}

// RemoveOrderResponse indicates whether the cancel order query was successful
type RemoveOrderResponse struct {
	ErrCapture `json:",omitempty"`
	Data       string `json:"data"`
}

// CreateOrderTempResponse stores orderID for successful orders creations
type CreateOrderTempResponse struct {
	OrderID int64 `json:"order_id"`
}

// CreateOrderResponse stores orderID for created orders
type CreateOrderResponse struct {
	ErrCapture `json:",omitempty"`
	Data       CreateOrderTempResponse `json:"data"`
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

// MarketDepthTemp stores market depth data
type MarketDepthTemp struct {
	Tick MarketDepth `json:"tick"`
}

// MarketDepthResponse stores market depth data
type MarketDepthResponse struct {
	ErrCapture `json:",omitempty"`
	Data       MarketDepthTemp `json:"data"`
}

// OpenOrderResponse stores data of all current delegation
type OpenOrderResponse struct {
	ErrCapture `json:",omitempty"`
	Data       OrderTempResponse `json:"data"`
}

// SelfTradeTempResponse stores self trade data
type SelfTradeTempResponse struct {
	OrderID int64 `json:"order_id"`
}

// SelfTradeResponse stores self trade data
type SelfTradeResponse struct {
	ErrCapture `json:",omitempty"`
	Data       SelfTradeTempResponse `json:"data"`
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
	Fee       float64 `json:"fee"`
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
	Price      float64 `json:"price"`
	Volume     string  `json:"volume"`
	DealVolume string  `json:"deal_volume"`
	TotalPrice float64 `json:"total_price"`
	Fee        float64 `json:"fee"`
	AvgPrice   float64 `json:"avg_price"`
}

// TradeData stores data for a trade
type TradeData struct {
	ID        int64   `json:"id"`
	CreatedAt string  `json:"created_at"`
	Price     float64 `json:"price"`
	Volume    string  `json:"volume"`
	DealPrice float64 `json:"deal_price"`
	Fee       float64 `json:"fee"`
}

// FetchOrderTempResp stores data for a given orderid
type FetchOrderTempResp struct {
	OrderInfo OrderData   `json:"order_info"`
	TradeList []TradeData `json:"trade_list"`
}

// FetchOrderResponse stores data for a given order id
type FetchOrderResponse struct {
	ErrCapture `json:",omitempty"`
	Data       FetchOrderTempResp `json:"data"`
}
