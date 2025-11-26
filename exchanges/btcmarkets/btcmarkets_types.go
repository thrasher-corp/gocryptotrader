package btcmarkets

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Market holds a tradable market instrument
type Market struct {
	MarketID       currency.Pair `json:"marketId"`
	BaseAsset      string        `json:"baseAssetName"`
	QuoteAsset     string        `json:"quoteAssetName"`
	MinOrderAmount float64       `json:"minOrderAmount,string"`
	MaxOrderAmount float64       `json:"maxOrderAmount,string"`
	AmountDecimals float64       `json:"amountDecimals,string"`
	PriceDecimals  float64       `json:"priceDecimals,string"`
	Status         string        `json:"status"`
}

// Ticker holds ticker information
type Ticker struct {
	MarketID  currency.Pair `json:"marketId"`
	BestBID   float64       `json:"bestBid,string"`
	BestAsk   float64       `json:"bestAsk,string"`
	LastPrice float64       `json:"lastPrice,string"`
	Volume    float64       `json:"volume24h,string"`
	Change24h float64       `json:"price24h,string"`
	Low24h    float64       `json:"low24h,string"`
	High24h   float64       `json:"high24h,string"`
	Timestamp time.Time     `json:"timestamp"`
}

// Trade holds trade information
type Trade struct {
	TradeID   string    `json:"id"`
	Amount    float64   `json:"amount,string"`
	Price     float64   `json:"price,string"`
	Timestamp time.Time `json:"timestamp"`
	Side      string    `json:"side"`
}

// tempOrderbook stores orderbook data
type tempOrderbook struct {
	MarketID   currency.Pair                    `json:"marketId"`
	SnapshotID int64                            `json:"snapshotId"`
	Asks       orderbook.LevelsArrayPriceAmount `json:"asks"`
	Bids       orderbook.LevelsArrayPriceAmount `json:"bids"`
}

// Orderbook holds current orderbook information returned from the exchange
type Orderbook struct {
	MarketID   currency.Pair
	SnapshotID int64
	Asks       []orderbook.Level
	Bids       []orderbook.Level
}

// MarketCandle stores candle data for a given pair
type MarketCandle struct {
	Time   time.Time
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
	AssetName currency.Code `json:"assetName"`
	Balance   float64       `json:"balance,string"`
	Available float64       `json:"available,string"`
	Locked    float64       `json:"locked,string"`
}

// TradeHistoryData stores data of past trades
type TradeHistoryData struct {
	ID            string        `json:"id"`
	MarketID      currency.Pair `json:"marketId"`
	Timestamp     time.Time     `json:"timestamp"`
	Price         float64       `json:"price,string"`
	Amount        float64       `json:"amount,string"`
	Side          string        `json:"side"`
	Fee           float64       `json:"fee,string"`
	OrderID       string        `json:"orderId"`
	LiquidityType string        `json:"liquidityType"`
}

// OrderData stores data for new order created
type OrderData struct {
	OrderID      string        `json:"orderId"`
	MarketID     currency.Pair `json:"marketId"`
	Side         string        `json:"side"`
	Type         string        `json:"type"`
	CreationTime time.Time     `json:"creationTime"`
	Price        float64       `json:"price,string"`
	Amount       float64       `json:"amount,string"`
	OpenAmount   float64       `json:"openAmount,string"`
	Status       string        `json:"status"`
	TargetAmount float64       `json:"targetAmount,string"`
	TimeInForce  string        `json:"timeInForce"`
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
	AssetName      currency.Code  `json:"assetName"`
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
	Tag       string // custom field we populate
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

// CreateReportResp stores data for created report
type CreateReportResp struct {
	ReportID string `json:"reportId"`
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
	OrderID       string        `json:"orderId"`
	MarketID      currency.Pair `json:"marketId"`
	Side          string        `json:"side"`
	Type          string        `json:"type"`
	CreationTime  time.Time     `json:"creationTime"`
	Price         float64       `json:"price,string"`
	Amount        float64       `json:"amount,string"`
	OpenAmount    float64       `json:"openAmount,string"`
	Status        string        `json:"status"`
	ClientOrderID string        `json:"clientOrderId"`
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

// CancelBatch stores data for batch cancel request
type CancelBatch struct {
	OrderID       string `json:"orderId,omitempty"`
	ClientOrderID string `json:"clientOrderId,omitempty"`
}

// PlaceBatch stores data for place batch request
type PlaceBatch struct {
	MarketID      string  `json:"marketId"`
	Price         float64 `json:"price"`
	Amount        float64 `json:"amount"`
	OrderType     string  `json:"type"`
	Side          string  `json:"side"`
	TriggerPrice  float64 `json:"triggerPrice,omitempty"`
	TriggerAmount float64 `json:"triggerAmount,omitempty"`
	TimeInForce   string  `json:"timeInForce,omitempty"`
	PostOnly      bool    `json:"postOnly,omitempty"`
	SelfTrade     string  `json:"selfTrade,omitempty"`
	ClientOrderID string  `json:"clientOrderId,omitempty"`
}

// PlaceOrderMethod stores data for place request
type PlaceOrderMethod struct {
	PlaceOrder PlaceBatch `json:"placeOrder"`
}

// CancelOrderMethod stores data for Cancel request
type CancelOrderMethod struct {
	CancelOrder CancelBatch `json:"cancelOrder"`
}

// TradingFeeData stores trading fee data
type TradingFeeData struct {
	MakerFeeRate float64       `json:"makerFeeRate,string"`
	TakerFeeRate float64       `json:"takerFeeRate,string"`
	MarketID     currency.Pair `json:"marketId"`
}

// TradingFeeResponse stores trading fee data
type TradingFeeResponse struct {
	MonthlyVolume float64          `json:"volume30Day,string"`
	FeeByMarkets  []TradingFeeData `json:"FeeByMarkets"`
}

// WsSubscribe defines a subscription message used in the Subscribe function
type WsSubscribe struct {
	MarketIDs   []string `json:"marketIds,omitempty"`
	Channels    []string `json:"channels,omitempty"`
	Key         string   `json:"key,omitempty"`
	Signature   string   `json:"signature,omitempty"`
	Timestamp   string   `json:"timestamp,omitempty"`
	MessageType string   `json:"messageType,omitempty"`
	ClientType  string   `json:"clientType,omitempty"`
}

// WsMessageType message sent via ws to determine type
type WsMessageType struct {
	MessageType string `json:"messageType"`
}

// WsTick message received for ticker data
type WsTick struct {
	MarketID    currency.Pair `json:"marketId"`
	Timestamp   time.Time     `json:"timestamp"`
	Bid         float64       `json:"bestBid,string"`
	Ask         float64       `json:"bestAsk,string"`
	Last        float64       `json:"lastPrice,string"`
	Volume      float64       `json:"volume24h,string"`
	Price24h    float64       `json:"price24h,string"`
	Low24h      float64       `json:"low24h,string"`
	High24      float64       `json:"high24h,string"`
	MessageType string        `json:"messageType"`
}

// WsTrade message received for trade data
type WsTrade struct {
	MarketID    currency.Pair `json:"marketId"`
	Timestamp   time.Time     `json:"timestamp"`
	TradeID     int64         `json:"tradeId"`
	Price       float64       `json:"price,string"`
	Volume      float64       `json:"volume,string"`
	Side        order.Side    `json:"side"`
	MessageType string        `json:"messageType"`
}

// WsOrderbook message received for orderbook data
type WsOrderbook struct {
	Currency    currency.Pair      `json:"marketId"`
	Snapshot    bool               `json:"snapshot"`
	Timestamp   time.Time          `json:"timestamp"`
	SnapshotID  int64              `json:"snapshotId"`
	Bids        WebsocketOrderbook `json:"bids"`
	Asks        WebsocketOrderbook `json:"asks"`
	Checksum    uint32             `json:"checksum,string"`
	MessageType string             `json:"messageType"`
}

// WsFundTransfer stores fund transfer data for websocket
type WsFundTransfer struct {
	FundTransferID int64     `json:"fundtransferId"`
	TransferType   string    `json:"type"`
	Status         string    `json:"status"`
	Timestamp      time.Time `json:"timestamp"`
	Amount         float64   `json:"amount,string"`
	Currency       string    `json:"currency"`
	Fee            float64   `json:"fee,string"`
	MessageType    string    `json:"messageType"`
}

// WsTradeData stores trade data for websocket
type WsTradeData struct {
	TradeID       int64   `json:"tradeId"`
	Price         float64 `json:"price,string"`
	Volume        float64 `json:"volume,string"`
	Fee           float64 `json:"fee,string"`
	LiquidityType string  `json:"liquidityType"`
}

// WsOrderChange stores order data
type WsOrderChange struct {
	OrderID       int64         `json:"orderId"`
	MarketID      currency.Pair `json:"marketId"`
	Side          string        `json:"side"`
	OrderType     string        `json:"type"`
	OpenVolume    float64       `json:"openVolume,string"`
	Status        string        `json:"status"`
	TriggerStatus string        `json:"triggerStatus"`
	Trades        []WsTradeData `json:"trades"`
	Timestamp     time.Time     `json:"timestamp"`
	MessageType   string        `json:"messageType"`
}

// WsError stores websocket error data
type WsError struct {
	MessageType string `json:"messageType"`
	Code        int64  `json:"code"`
	Message     string `json:"message"`
}

// CandleResponse holds OHLCV data for exchange
type CandleResponse struct {
	Timestamp time.Time
	Open      types.Number
	High      types.Number
	Low       types.Number
	Close     types.Number
	Volume    types.Number
}

// UnmarshalJSON unmarshals the CandleResponse struct from JSON data.
func (c *CandleResponse) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[6]any{&c.Timestamp, &c.Open, &c.High, &c.Low, &c.Close, &c.Volume})
}

// WebsocketOrderbook defines a specific websocket orderbook type to directly
// unmarshal json.
type WebsocketOrderbook orderbook.Levels
