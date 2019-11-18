package btcmarkets

import "time"

// Market holds a tradable market instrument
type Market struct {
	MarketID       string  `json:"marketId"`
	BaseAsset      string  `json:"baseAsset"`
	QuoteAsset     string  `json:"quoteAsset"`
	MinOrderAmount float64 `json:"minOrderAmount,string"`
	MaxOrderAmount float64 `json:"maxOrderAmount,string"`
	AmountDecimals int64   `json:"amountDecimals,string"`
	PriceDecimals  int64   `json:"priceDecimals,string"`
}

// Ticker holds ticker information
type Ticker struct {
	MarketID  string    `json:"marketId"`
	BestBID   float64   `json:"bestBid,string"`
	BestAsk   float64   `json:"bestAsk,string"`
	LastPrice float64   `json:"lastPrice,string"`
	Volume    float64   `json:"volume24h,string"`
	Change24h float64   `json:"price24h,string"`
	Low24h    float64   `json:"low24h,string"`
	High24h   float64   `json:"high24h,string"`
	Timestamp time.Time `json:"timestamp"`
}

// Trade holds trade information
type Trade struct {
	TradeID   string    `json:"id"`
	Amount    float64   `json:"amount,string"`
	Price     float64   `json:"price,string"`
	Timestamp time.Time `json:"timestamp"`
}

// tempOrderbook stores orderbook data
type tempOrderbook struct {
	MarketID   string      `json:"marketId"`
	SnapshotID int64       `json:"snapshotId"`
	Asks       [][2]string `json:"asks"`
	Bids       [][2]string `json:"bids"`
}

// OBData stores orderbook data
type OBData struct {
	Price  float64
	Volume float64
}

// Orderbook holds current orderbook information returned from the exchange
type Orderbook struct {
	MarketID   string
	SnapshotID int64
	Asks       []OBData
	Bids       []OBData
}

// MarketCandle stores candle data for a given pair
type MarketCandle struct {
	Time   string
	Open   float64
	Close  float64
	Low    float64
	High   float64
	Volume float64
}

// TimeResp stores server time
type TimeResp struct {
	Time time.Time `json:"timestamp"`
}

// TradingFee 30 day trade volume
type TradingFee struct {
	Success        bool    `json:"success"`
	ErrorCode      int     `json:"errorCode"`
	ErrorMessage   string  `json:"errorMessage"`
	TradingFeeRate float64 `json:"tradingfeerate"`
	Volume30Day    float64 `json:"volume30day"`
}

// OrderToGo holds order information to be sent to the exchange
type OrderToGo struct {
	Currency        string `json:"currency"`
	Instrument      string `json:"instrument"`
	Price           int64  `json:"price"`
	Volume          int64  `json:"volume"`
	OrderSide       string `json:"orderSide"`
	OrderType       string `json:"ordertype"`
	ClientRequestID string `json:"clientRequestId"`
}

// Order holds order information
type Order struct {
	ID              int64           `json:"id"`
	Currency        string          `json:"currency"`
	Instrument      string          `json:"instrument"`
	OrderSide       string          `json:"orderSide"`
	OrderType       string          `json:"ordertype"`
	CreationTime    time.Time       `json:"creationTime"`
	Status          string          `json:"status"`
	ErrorMessage    string          `json:"errorMessage"`
	Price           float64         `json:"price"`
	Volume          float64         `json:"volume"`
	OpenVolume      float64         `json:"openVolume"`
	ClientRequestID string          `json:"clientRequestId"`
	Trades          []TradeResponse `json:"trades"`
}

// TradeResponse holds trade information
type TradeResponse struct {
	ID           int64     `json:"id"`
	CreationTime time.Time `json:"creationTime"`
	Description  string    `json:"description"`
	Price        float64   `json:"price"`
	Volume       float64   `json:"volume"`
	Fee          float64   `json:"fee"`
}

// AccountData stores account data
type AccountData struct {
	AssetName string  `json:"assetName"`
	Balance   float64 `json:"balance,string"`
	Available float64 `json:"available,string"`
	Locked    float64 `json:"locked,string"`
}

// TradeHistoryData stores data of past trades
type TradeHistoryData struct {
	ID            string  `json:"id"`
	MarketID      string  `json:"marketId"`
	Timestamp     string  `json:"timestamp"`
	Price         float64 `json:"price,string"`
	Amount        float64 `json:"amount,string"`
	Side          string  `json:"side"`
	Fee           float64 `json:"fee,string"`
	OrderID       string  `json:"orderId"`
	LiquidityType string  `json:"liquidityType"`
}

// OrderData stores data for new order created
type OrderData struct {
	OrderID      string    `json:"orderId"`
	MarketID     string    `json:"marketId"`
	Side         string    `json:"side"`
	Type         string    `json:"type"`
	CreationTime time.Time `json:"creationTime"`
	Price        float64   `json:"price,string"`
	Amount       float64   `json:"amount,string"`
	OpenAmount   float64   `json:"openAmount,string"`
	Status       string    `json:"status"`
}

// CancelOrderResp stores data for cancelled orders
type CancelOrderResp struct {
	OrderID       string `json:"orderId"`
	ClientOrderID string `json:"clientOrderId"`
}

// PaymentDetails stores payment address
type PaymentDetails struct {
	Address string `json:"address"`
}

// TransferData stores data from asset transfers
type TransferData struct {
	ID             string         `json:"id"`
	AssetName      string         `json:"assetName"`
	Amount         float64        `json:"amount,string"`
	RequestType    string         `json:"type"`
	CreationTime   time.Time      `json:"creationTime"`
	Status         string         `json:"status"`
	Description    string         `json:"description"`
	Fee            float64        `json:"fee,string"`
	LastUpdate     string         `json:"lastUpdate"`
	PaymentDetails PaymentDetails `json:"paymentDetail"`
}

// DepositAddress stores deposit address data
type DepositAddress struct {
	Address   string `json:"address"`
	AssetName string `json:"assetName"`
}

// WithdrawalFeeData stores data for fees
type WithdrawalFeeData struct {
	AssetName string  `json:"assetName"`
	Fee       float64 `json:"fee,string"`
}

// AssetData stores data for given asset
type AssetData struct {
	AssetName           string  `json:"assetName"`
	MinDepositAmount    float64 `json:"minDepositAmount,string"`
	MaxDepositAmount    float64 `json:"maxDepositAmount,string"`
	DepositDecimals     float64 `json:"depositDecimals,string"`
	MinWithdrawalAmount float64 `json:"minWithdrawalAmount,string"`
	MaxWithdrawalAmount float64 `json:"maxWithdrawalAmount,string"`
	WithdrawalDecimals  float64 `json:"withdrawalDecimals,string"`
	WithdrawalFee       float64 `json:"withdrawalFee,string"`
	DepositFee          float64 `json:"depositFee,string"`
}

// TransactionData stores data from past transactions
type TransactionData struct {
	ID           string    `json:"id"`
	CreationTime time.Time `json:"creationTime"`
	Description  string    `json:"description"`
	AssetName    string    `json:"assetName"`
	Amount       float64   `json:"amount,string"`
	Balance      float64   `json:"balance,string"`
	FeeType      string    `json:"type"`
	RecordType   string    `json:"recordType"`
	ReferrenceID string    `json:"referrenceId"`
}

// ReportData gets data for a created report
type ReportData struct {
	ID           string    `json:"id"`
	ContentURL   string    `json:"contentUrl"`
	CreationTime time.Time `json:"creationTime"`
	ReportType   string    `json:"reportType"`
	Status       string    `json:"status"`
	Format       string    `json:"format"`
}

// BatchPlaceData stores data for placed batch orders
type BatchPlaceData struct {
	OrderID       string    `json:"orderId"`
	MarketID      string    `json:"marketId"`
	Side          string    `json:"side"`
	Type          string    `json:"type"`
	CreationTime  time.Time `json:"creationTime"`
	Price         float64   `json:"price,string"`
	Amount        float64   `json:"amount,string"`
	OpenAmount    float64   `json:"openAmount,string"`
	Status        string    `json:"status"`
	ClientOrderID string    `json:"clientOrderId"`
}

// UnprocessedBatchResp stores data for unprocessed response
type UnprocessedBatchResp struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"requestId"`
}

// BatchPlaceCancelResponse stores place and cancel batch data
type BatchPlaceCancelResponse struct {
	PlacedOrders      []BatchPlaceData       `json:"placeOrders"`
	CancelledOrders   []CancelOrderResp      `json:"cancelOrders"`
	UnprocessedOrders []UnprocessedBatchResp `json:"unprocessedRequests"`
}

// BatchTradeResponse stores the trades from batchtrades
type BatchTradeResponse struct {
	Orders              []BatchPlaceData       `json:"orders"`
	UnprocessedRequests []UnprocessedBatchResp `json:"unprocessedRequests"`
}

// BatchCancelResponse stores the cancellation details from batch cancels
type BatchCancelResponse struct {
	CancelOrders        []CancelOrderResp      `json:"cancelOrders"`
	UnprocessedRequests []UnprocessedBatchResp `json:"unprocessedRequests"`
}

// WithdrawRequestCrypto is a generalized withdraw request type
type WithdrawRequestCrypto struct {
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
	Address  string `json:"address"`
}

// WithdrawRequestAUD is a generalized withdraw request type
type WithdrawRequestAUD struct {
	Amount        int64  `json:"amount"`
	Currency      string `json:"currency"`
	AccountName   string `json:"accountName"`
	AccountNumber string `json:"accountNumber"`
	BankName      string `json:"bankName"`
	BSBNumber     string `json:"bsbNumber"`
}

// CancelBatch stores data from batch cancel request
type CancelBatch struct {
	OrderID       string `json:"orderId"`
	ClientOrderID string `json:"clientOrderId"`
}

// PlaceBatch stores data from place batch request
type PlaceBatch struct {
	MarketID      string  `json:"marketId"`
	Price         float64 `json:"price"`
	Amount        float64 `json:"amount"`
	OrderType     string  `json:"type"`
	Side          string  `json:"side"`
	TriggerPrice  float64 `json:"triggerPrice"`
	TriggerAmount float64 `json:"triggerAmount"`
	TimeInForce   string  `json:"timeInForce"`
	PostOnly      bool    `json:"postOnly"`
	SelfTrade     string  `json:"selfTrade"`
	ClientOrderID string  `json:"clientOrderId"`
}

// TradingFeeData stores trading fee data
type TradingFeeData struct {
	MakerFeeRate float64 `json:"makerFeeRate,string"`
	TakerFeeRate float64 `json:"takerFeeRate,string"`
	MarketID     string  `json:"marketId"`
}

// TradingFeeResponse stores trading fee data
type TradingFeeResponse struct {
	MonthlyVolume float64          `json:"volume30Day,string"`
	FeeByMarkets  []TradingFeeData `json:"FeeByMarkets"`
}

// WsSubscribe message sent via ws to subscribe
type WsSubscribe struct {
	MarketIDs   []string `json:"marketIds,omitempty"`
	Channels    []string `json:"channels"`
	MessageType string   `json:"messageType"`
}

// WsMessageType message sent via ws to determine type
type WsMessageType struct {
	MessageType string `json:"messageType"`
}

// WsTick message received for ticker data
type WsTick struct {
	Currency    string    `json:"marketId"`
	Timestamp   time.Time `json:"timestamp"`
	Bid         float64   `json:"bestBid,string"`
	Ask         float64   `json:"bestAsk,string"`
	Last        float64   `json:"lastPrice,string"`
	Volume      float64   `json:"volume24h,string"`
	Price24h    float64   `json:"price24h,string"`
	Low24h      float64   `json:"low24h,string"`
	High24      float64   `json:"high24h,string"`
	MessageType string    `json:"messageType"`
}

// WsTrade message received for trade data
type WsTrade struct {
	Currency    string    `json:"marketId"`
	Timestamp   time.Time `json:"timestamp"`
	TradeID     int64     `json:"tradeId"`
	Price       float64   `json:"price,string"`
	Volume      float64   `json:"volume,string"`
	MessageType string    `json:"messageType"`
}

// WsOrderbook message received for orderbook data
type WsOrderbook struct {
	Currency    string     `json:"marketId"`
	Timestamp   time.Time  `json:"timestamp"`
	Bids        [][]string `json:"bids"`
	Asks        [][]string `json:"asks"`
	MessageType string     `json:"messageType"`
}

// WsError stores websocket error data
type WsError struct {
	MessageType string `json:"messageType"`
	Code        int64  `json:"code"`
	Message     string `json:"message"`
}
