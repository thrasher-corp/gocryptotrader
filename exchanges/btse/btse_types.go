package btse

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

const (
	// Default order type is good till cancel (or filled)
	goodTillCancel = "GTC"

	orderInserted  = 2
	orderCancelled = 6
)

// FundingHistoryData stores funding history data
type FundingHistoryData struct {
	Time   types.Time `json:"time"`
	Rate   float64    `json:"rate"`
	Symbol string     `json:"symbol"`
}

// MarketSummary response data
type MarketSummary []*MarketPair

// MarketPair is a single pair in Market Summary
type MarketPair struct {
	Symbol              string     `json:"symbol"`
	Last                float64    `json:"last"`
	LowestAsk           float64    `json:"lowestAsk"`
	HighestBid          float64    `json:"highestBid"`
	PercentageChange    float64    `json:"percentageChange"`
	Volume              float64    `json:"volume"`
	High24Hr            float64    `json:"high24Hr"`
	Low24Hr             float64    `json:"low24Hr"`
	Base                string     `json:"base"`
	Quote               string     `json:"quote"`
	Active              bool       `json:"active"`
	Size                float64    `json:"size"`
	MinValidPrice       float64    `json:"minValidPrice"`
	MinPriceIncrement   float64    `json:"minPriceIncrement"`
	MinOrderSize        float64    `json:"minOrderSize"`
	MaxOrderSize        float64    `json:"maxOrderSize"`
	MinSizeIncrement    float64    `json:"minSizeIncrement"`
	OpenInterest        float64    `json:"openInterest"`
	OpenInterestUSD     float64    `json:"openInterestUSD"`
	ContractStart       int64      `json:"contractStart"`
	ContractEnd         int64      `json:"contractEnd"`
	TimeBasedContract   bool       `json:"timeBasedContract"`
	OpenTime            types.Time `json:"openTime"`
	CloseTime           types.Time `json:"closeTime"`
	StartMatching       int64      `json:"startMatching"`
	InactiveTime        types.Time `json:"inactiveTime"`
	FundingRate         float64    `json:"fundingRate"`
	ContractSize        float64    `json:"contractSize"`
	MaxPosition         int64      `json:"maxPosition"`
	MinRiskLimit        int        `json:"minRiskLimit"`
	MaxRiskLimit        int        `json:"maxRiskLimit"`
	AvailableSettlement []string   `json:"availableSettlement"`
	Futures             bool       `json:"futures"`
	IsMarketOpenToSpot  bool       `json:"isMarketOpenToSpot"`
	IsMarketOpenToOTC   bool       `json:"isMarketOpenToOtc"`
}

// OHLCV holds Open, High Low, Close, Volume data for set symbol
type OHLCV [][]float64

// Price stores last price for requested symbol
type Price []struct {
	IndexPrice float64 `json:"indexPrice"`
	LastPrice  float64 `json:"lastPrice"`
	MarkPrice  float64 `json:"markPrice"`
	Symbol     string  `json:"symbol"`
}

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
	Symbol              string     `json:"symbol"`
	Last                float64    `json:"last"`
	LowestAsk           float64    `json:"lowestAsk"`
	HighestBid          float64    `json:"highestBid"`
	OpenInterest        float64    `json:"openInterest"`
	OpenInterestUSD     float64    `json:"openInterestUSD"`
	PercentageChange    float64    `json:"percentageChange"`
	Volume              float64    `json:"volume"`
	High24Hr            float64    `json:"high24Hr"`
	Low24Hr             float64    `json:"low24Hr"`
	Base                string     `json:"base"`
	Quote               string     `json:"quote"`
	ContractStart       int64      `json:"contractStart"`
	ContractEnd         int64      `json:"contractEnd"`
	Active              bool       `json:"active"`
	TimeBasedContract   bool       `json:"timeBasedContract"`
	OpenTime            types.Time `json:"openTime"`
	CloseTime           types.Time `json:"closeTime"`
	StartMatching       types.Time `json:"startMatching"`
	InactiveTime        types.Time `json:"inactiveTime"`
	FundingRate         float64    `json:"fundingRate"`
	ContractSize        float64    `json:"contractSize"`
	MaxPosition         int64      `json:"maxPosition"`
	MinValidPrice       float64    `json:"minValidPrice"`
	MinPriceIncrement   float64    `json:"minPriceIncrement"`
	MinOrderSize        int32      `json:"minOrderSize"`
	MaxOrderSize        int32      `json:"maxOrderSize"`
	MinRiskLimit        int32      `json:"minRiskLimit"`
	MaxRiskLimit        int32      `json:"maxRiskLimit"`
	MinSizeIncrement    float64    `json:"minSizeIncrement"`
	AvailableSettlement []string   `json:"availableSettlement"`
}

// Trade stores trade data
type Trade struct {
	SerialID int64      `json:"serialId"`
	Symbol   string     `json:"symbol"`
	Price    float64    `json:"price"`
	Amount   float64    `json:"size"`
	Time     types.Time `json:"timestamp"`
	Side     string     `json:"side"`
	Type     string     `json:"type"`
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
	Timestamp types.Time  `json:"timestamp"`
}

// Ticker stores the ticker data
type Ticker struct {
	Price  float64    `json:"price,string"`
	Size   float64    `json:"size,string"`
	Bid    float64    `json:"bid,string"`
	Ask    float64    `json:"ask,string"`
	Volume float64    `json:"volume,string"`
	Time   types.Time `json:"time"`
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
	Epoch int64     `json:"epoch"`
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
	Base         string     `json:"base"`
	ClOrderID    string     `json:"clOrderID"`
	FeeAmount    float64    `json:"feeAmount"`
	FeeCurrency  string     `json:"feeCurrency"`
	FilledPrice  float64    `json:"filledPrice"`
	FilledSize   float64    `json:"filledSize"`
	OrderID      string     `json:"orderId"`
	OrderType    int        `json:"orderType"`
	Price        float64    `json:"price"`
	Quote        string     `json:"quote"`
	RealizedPnl  float64    `json:"realizedPnl"`
	SerialID     int64      `json:"serialId"`
	Side         string     `json:"side"`
	Size         float64    `json:"size"`
	Symbol       string     `json:"symbol"`
	Timestamp    types.Time `json:"timestamp"`
	Total        float64    `json:"total"`
	TradeID      string     `json:"tradeId"`
	TriggerPrice float64    `json:"triggerPrice"`
	TriggerType  int        `json:"triggerType"`
	Username     string     `json:"username"`
	Wallet       string     `json:"wallet"`
}

// WalletHistory stores account funding history
type WalletHistory []struct {
	Amount      float64    `json:"amount"`
	Currency    string     `json:"currency"`
	Description string     `json:"description"`
	Fees        float64    `json:"fees"`
	OrderID     string     `json:"orderId"`
	Status      string     `json:"status"`
	Timestamp   types.Time `json:"timestamp"`
	Type        string     `json:"type"`
	Username    string     `json:"username"`
	Wallet      string     `json:"wallet"`
}

// WalletAddress stores address for crypto deposit's
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
	AverageFillPrice             float64    `json:"averageFillPrice"`
	CancelDuration               int64      `json:"cancelDuration"`
	ClOrderID                    string     `json:"clOrderID"`
	FillSize                     float64    `json:"fillSize"`
	FilledSize                   float64    `json:"filledSize"`
	OrderID                      string     `json:"orderID"`
	OrderState                   string     `json:"orderState"`
	OrderType                    int        `json:"orderType"`
	OrderValue                   float64    `json:"orderValue"`
	PegPriceDeviation            float64    `json:"pegPriceDeviation"`
	PegPriceMax                  float64    `json:"pegPriceMax"`
	PegPriceMin                  float64    `json:"pegPriceMin"`
	Price                        float64    `json:"price"`
	Side                         string     `json:"side"`
	Size                         float64    `json:"size"`
	Symbol                       string     `json:"symbol"`
	Timestamp                    types.Time `json:"timestamp"`
	TrailValue                   float64    `json:"trailValue"`
	TriggerOrder                 bool       `json:"triggerOrder"`
	TriggerOrderType             int        `json:"triggerOrderType"`
	TriggerOriginalPrice         float64    `json:"triggerOriginalPrice"`
	TriggerPrice                 float64    `json:"triggerPrice"`
	TriggerStopPrice             float64    `json:"triggerStopPrice"`
	TriggerTrailingStopDeviation float64    `json:"triggerTrailingStopDeviation"`
	Triggered                    bool       `json:"triggered"`
}

// CancelOrder stores slice of orders
type CancelOrder []Order

// Order stores information for a single order
type Order struct {
	AverageFillPrice float64    `json:"averageFillPrice"`
	ClOrderID        string     `json:"clOrderID"`
	Deviation        float64    `json:"deviation"`
	FillSize         float64    `json:"fillSize"`
	Message          string     `json:"message"`
	OrderID          string     `json:"orderID"`
	OrderType        int        `json:"orderType"`
	Price            float64    `json:"price"`
	Side             string     `json:"side"`
	Size             float64    `json:"size"`
	Status           int        `json:"status"`
	Stealth          float64    `json:"stealth"`
	StopPrice        float64    `json:"stopPrice"`
	Symbol           string     `json:"symbol"`
	Timestamp        types.Time `json:"timestamp"`
	Trigger          bool       `json:"trigger"`
	TriggerPrice     float64    `json:"triggerPrice"`
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
	Symbol    string     `json:"symbol"`
	Side      order.Side `json:"side"`
	Size      float64    `json:"size"`
	Price     float64    `json:"price"`
	TID       int64      `json:"tradeID"`
	Timestamp types.Time `json:"timestamp"`
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
	OrderID           string     `json:"orderID"`
	OrderMode         string     `json:"orderMode"`
	OrderType         string     `json:"orderType"`
	PegPriceDeviation string     `json:"pegPriceDeviation"`
	Price             float64    `json:"price,string"`
	Size              float64    `json:"size,string"`
	Status            string     `json:"status"`
	Stealth           string     `json:"stealth"`
	Symbol            string     `json:"symbol"`
	Timestamp         types.Time `json:"timestamp"`
	TriggerPrice      float64    `json:"triggerPrice,string"`
	Type              string     `json:"type"`
}

// ErrorResponse contains errors received from API
type ErrorResponse struct {
	ErrorCode int    `json:"errorCode"`
	Message   string `json:"message"`
	Status    int    `json:"status"`
}

// WsSubscriptionAcknowledgement contains successful subscription messages
type WsSubscriptionAcknowledgement struct {
	Channel []string `json:"channel"`
	Event   string   `json:"event"`
}

// WsLoginAcknowledgement contains whether authentication was successful
type WsLoginAcknowledgement struct {
	Event   string `json:"event"`
	Success bool   `json:"success"`
}
