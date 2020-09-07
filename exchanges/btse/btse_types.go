package btse

import "time"

const (
	// Default order type is good till cancel (or filled)
	goodTillCancel = "GTC"
)

// MarketSummary response data
type MarketSummary []struct {
	Symbol              string      `json:"symbol"`
	Last                float64     `json:"last"`
	LowestAsk           float64     `json:"lowestAsk"`
	HighestBid          float64     `json:"highestBid"`
	PercentageChange    float64     `json:"percentageChange"`
	Volume              float64     `json:"volume"`
	High24Hr            float64     `json:"high24Hr"`
	Low24Hr             float64     `json:"low24Hr"`
	Base                string      `json:"base"`
	Quote               string      `json:"quote"`
	Active              bool        `json:"active"`
	Size                float64     `json:"size"`
	MinValidPrice       float64     `json:"minValidPrice"`
	MinPriceIncrement   float64     `json:"minPriceIncrement"`
	MinOrderSize        float64     `json:"minOrderSize"`
	MaxOrderSize        float64     `json:"maxOrderSize"`
	MinSizeIncrement    float64     `json:"minSizeIncrement"`
	OpenInterest        float64     `json:"openInterest"`
	OpenInterestUSD     float64     `json:"openInterestUSD"`
	ContractStart       int         `json:"contractStart"`
	ContractEnd         int         `json:"contractEnd"`
	TimeBasedContract   bool        `json:"timeBasedContract"`
	OpenTime            int         `json:"openTime"`
	CloseTime           int         `json:"closeTime"`
	StartMatching       int         `json:"startMatching"`
	InactiveTime        int         `json:"inactiveTime"`
	FundingRate         float64     `json:"fundingRate"`
	ContractSize        float64     `json:"contractSize"`
	MaxPosition         int         `json:"maxPosition"`
	MinRiskLimit        int         `json:"minRiskLimit"`
	MaxRiskLimit        int         `json:"maxRiskLimit"`
	AvailableSettlement interface{} `json:"availableSettlement"`
	Futures             bool        `json:"futures"`
}

// OHLCV holds OHLCV data for set symbol
type OHLCV [][]float64

// SpotMarket stores market data
type SpotMarket struct {
	Symbol            string  `json:"symbol"`
	ID                string  `json:"id"`
	BaseCurrency      string  `json:"base_currency"`
	QuoteCurrency     string  `json:"quote_currency"`
	BaseMinSize       float64 `json:"base_min_size"`
	BaseMaxSize       float64 `json:"base_max_size"`
	BaseIncrementSize float64 `json:"base_increment_size"`
	QuoteMinPrice     float64 `json:"quote_min_price"`
	QuoteIncrement    float64 `json:"quote_increment"`
	Status            string  `json:"status"`
}

// FuturesMarket stores market data
type FuturesMarket struct {
	Symbol              string   `json:"symbol"`
	Last                float64  `json:"last"`
	LowestAsk           float64  `json:"lowestAsk"`
	HighestBid          float64  `json:"highestBid"`
	OpenInterest        float64  `json:"openInterest"`
	OpenInterestUSD     float64  `json:"openInterestUSD"`
	PercentageChange    float64  `json:"percentageChange"`
	Volume              float64  `json:"volume"`
	High24Hr            float64  `json:"high24Hr"`
	Low24Hr             float64  `json:"low24Hr"`
	Base                string   `json:"base"`
	Quote               string   `json:"quote"`
	ContractStart       int64    `json:"contractStart"`
	ContractEnd         int64    `json:"contractEnd"`
	Active              bool     `json:"active"`
	TimeBasedContract   bool     `json:"timeBasedContract"`
	OpenTime            int64    `json:"openTime"`
	CloseTime           int64    `json:"closeTime"`
	StartMatching       int64    `json:"startMatching"`
	InactiveTime        int64    `json:"inactiveTime"`
	FundingRate         float64  `json:"fundingRate"`
	ContractSize        float64  `json:"contractSize"`
	MaxPosition         int64    `json:"maxPosition"`
	MinValidPrice       float64  `json:"minValidPrice"`
	MinPriceIncrement   float64  `json:"minPriceIncrement"`
	MinOrderSize        int32    `json:"minOrderSize"`
	MaxOrderSize        int32    `json:"maxOrderSize"`
	MinRiskLimit        int32    `json:"minRiskLimit"`
	MaxRiskLimit        int32    `json:"maxRiskLimit"`
	MinSizeIncrement    float64  `json:"minSizeIncrement"`
	AvailableSettlement []string `json:"availableSettlement"`
}

// Trade stores trade data
type Trade struct {
	SerialID string    `json:"serial_id"`
	Symbol   string    `json:"symbol"`
	Price    float64   `json:"price"`
	Amount   float64   `json:"amount"`
	Time     time.Time `json:"time"`
	Type     string    `json:"type"`
}

// QuoteData stores quote data
type QuoteData struct {
	Price float64 `json:"price,string"`
	Size  float64 `json:"size,string"`
}

// Orderbook stores orderbook info
type Orderbook struct {
	BuyQuote  []QuoteData `json:"buyQuote"`
	SellQuote []QuoteData `json:"sellQuote"`
	Symbol    string      `json:"symbol"`
	Timestamp int64       `json:"timestamp"`
}

// Ticker stores the ticker data
type Ticker struct {
	Price  float64 `json:"price,string"`
	Size   float64 `json:"size,string"`
	Bid    float64 `json:"bid,string"`
	Ask    float64 `json:"ask,string"`
	Volume float64 `json:"volume,string"`
	Time   string  `json:"time"`
}

// MarketStatistics stores market statistics for a particular product
type MarketStatistics struct {
	Open   float64   `json:"open,string"`
	Low    float64   `json:"low,string"`
	High   float64   `json:"high,string"`
	Close  float64   `json:"close,string"`
	Volume float64   `json:"volume,string"`
	Time   time.Time `json:"time"`
}

// ServerTime stores the server time data
type ServerTime struct {
	ISO   time.Time `json:"iso"`
	Epoch int       `json:"epoch"`
}

// CurrencyBalance stores the account info data
type CurrencyBalance struct {
	Currency  string  `json:"currency"`
	Total     float64 `json:"total"`
	Available float64 `json:"available"`
}

// AccountFees stores fee for each currency pair
type AccountFees struct {
	MakerFee float64 `json:"makerFee"`
	Symbol   string  `json:"symbol"`
	TakerFee float64 `json:"takerFee"`
}

// TradeHistory stores user trades for exchange
type TradeHistory []struct {
	Base         string  `json:"base"`
	ClOrderID    string  `json:"clOrderID"`
	FeeAmount    float64 `json:"feeAmount"`
	FeeCurrency  string  `json:"feeCurrency"`
	FilledPrice  int     `json:"filledPrice"`
	FilledSize   int     `json:"filledSize"`
	OrderID      string  `json:"orderId"`
	OrderType    int     `json:"orderType"`
	Price        float64 `json:"price"`
	Quote        string  `json:"quote"`
	RealizedPnl  int     `json:"realizedPnl"`
	SerialID     int     `json:"serialId"`
	Side         string  `json:"side"`
	Size         float64 `json:"size"`
	Symbol       string  `json:"symbol"`
	Timestamp    string  `json:"timestamp"`
	Total        int     `json:"total"`
	TradeID      string  `json:"tradeId"`
	TriggerPrice int     `json:"triggerPrice"`
	TriggerType  int     `json:"triggerType"`
	Username     string  `json:"username"`
	Wallet       string  `json:"wallet"`
}

// WalletHistory stores account funding history
type WalletHistory []struct {
	Amount      float64 `json:"amount"`
	Currency    string  `json:"currency"`
	Description string  `json:"description"`
	Fees        float64 `json:"fees"`
	OrderID     int64   `json:"orderId"`
	Status      int     `json:"status"`
	Timestamp   int64   `json:"timestamp"`
	Type        int     `json:"type"`
	Username    string  `json:"username"`
	Wallet      string  `json:"wallet"`
}

// WalletAddress stores address's for deposit's
type WalletAddress []struct {
	Address string `json:"address"`
	Created int    `json:"created"`
}

// WithdrawalResponse response received when submitting a crypto withdrawal request
type WithdrawalResponse struct {
	WithdrawID string `json:"withdraw_id"`
}

// OpenOrder stores an open order info
type OpenOrder struct {
	ID         string  `json:"id"`
	Type       string  `json:"type"`
	Side       string  `json:"side"`
	Price      float64 `json:"price"`
	Size       float64 `json:"size"`
	Tag        string  `json:"tag"`
	Symbol     string  `json:"symbol"`
	CreatedAt  string  `json:"created_at"`
	OrderState string  `json:"orderState"`
}

type CancelOrder []struct {
	AverageFillPrice int    `json:"averageFillPrice"`
	ClOrderID        string `json:"clOrderID"`
	Deviation        int    `json:"deviation"`
	FillSize         int    `json:"fillSize"`
	Message          string `json:"message"`
	OrderID          string `json:"orderID"`
	OrderType        int    `json:"orderType"`
	Price            int    `json:"price"`
	Side             string `json:"side"`
	Size             int    `json:"size"`
	Status           int    `json:"status"`
	Stealth          int    `json:"stealth"`
	StopPrice        int    `json:"stopPrice"`
	Symbol           string `json:"symbol"`
	Timestamp        int64  `json:"timestamp"`
	Trigger          bool   `json:"trigger"`
	TriggerPrice     int    `json:"triggerPrice"`
}

type wsSub struct {
	Operation string   `json:"op"`
	Arguments []string `json:"args"`
}

type wsQuoteData struct {
	Total string `json:"cumulativeTotal"`
	Price string `json:"price"`
	Size  string `json:"size"`
}

type wsOBData struct {
	Currency  string        `json:"currency"`
	BuyQuote  []wsQuoteData `json:"buyQuote"`
	SellQuote []wsQuoteData `json:"sellQuote"`
}

type wsOrderBook struct {
	Topic string   `json:"topic"`
	Data  wsOBData `json:"data"`
}

type wsTradeData struct {
	Amount          float64 `json:"amount"`
	Gain            int64   `json:"gain"`
	Newest          int64   `json:"newest"`
	Price           float64 `json:"price"`
	ID              int64   `json:"serialId"`
	TransactionTime int64   `json:"transactionUnixTime"`
}

type wsTradeHistory struct {
	Topic string        `json:"topic"`
	Data  []wsTradeData `json:"data"`
}

type wsNotification struct {
	Topic string          `json:"topic"`
	Data  []wsOrderUpdate `json:"data"`
}

type wsOrderUpdate struct {
	OrderID           string  `json:"orderID"`
	OrderMode         string  `json:"orderMode"`
	OrderType         string  `json:"orderType"`
	PegPriceDeviation string  `json:"pegPriceDeviation"`
	Price             float64 `json:"price,string"`
	Size              float64 `json:"size,string"`
	Status            string  `json:"status"`
	Stealth           string  `json:"stealth"`
	Symbol            string  `json:"symbol"`
	Timestamp         int64   `json:"timestamp,string"`
	TriggerPrice      float64 `json:"triggerPrice,string"`
	Type              string  `json:"type"`
}

// ErrorResponse contains errors received from api
type ErrorResponse struct {
	ErrorCode int    `json:"errorCode"`
	Message   string `json:"message"`
	Status    int    `json:"status"`
}
