package binance

import (
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/bank"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fee"
)

const wsRateLimitMilliseconds = 250

// withdrawals status codes description
const (
	EmailSent = iota
	Cancelled
	AwaitingApproval
	Rejected
	Processing
	Failure
	Completed
)

// ExchangeInfo holds the full exchange information type
type ExchangeInfo struct {
	Code       int       `json:"code"`
	Msg        string    `json:"msg"`
	Timezone   string    `json:"timezone"`
	Servertime time.Time `json:"serverTime"`
	RateLimits []struct {
		RateLimitType string `json:"rateLimitType"`
		Interval      string `json:"interval"`
		Limit         int    `json:"limit"`
	} `json:"rateLimits"`
	ExchangeFilters interface{} `json:"exchangeFilters"`
	Symbols         []struct {
		Symbol                     string   `json:"symbol"`
		Status                     string   `json:"status"`
		BaseAsset                  string   `json:"baseAsset"`
		BaseAssetPrecision         int      `json:"baseAssetPrecision"`
		QuoteAsset                 string   `json:"quoteAsset"`
		QuotePrecision             int      `json:"quotePrecision"`
		OrderTypes                 []string `json:"orderTypes"`
		IcebergAllowed             bool     `json:"icebergAllowed"`
		OCOAllowed                 bool     `json:"ocoAllowed"`
		QuoteOrderQtyMarketAllowed bool     `json:"quoteOrderQtyMarketAllowed"`
		IsSpotTradingAllowed       bool     `json:"isSpotTradingAllowed"`
		IsMarginTradingAllowed     bool     `json:"isMarginTradingAllowed"`
		Filters                    []struct {
			FilterType          string  `json:"filterType"`
			MinPrice            float64 `json:"minPrice,string"`
			MaxPrice            float64 `json:"maxPrice,string"`
			TickSize            float64 `json:"tickSize,string"`
			MultiplierUp        float64 `json:"multiplierUp,string"`
			MultiplierDown      float64 `json:"multiplierDown,string"`
			AvgPriceMinutes     int64   `json:"avgPriceMins"`
			MinQty              float64 `json:"minQty,string"`
			MaxQty              float64 `json:"maxQty,string"`
			StepSize            float64 `json:"stepSize,string"`
			MinNotional         float64 `json:"minNotional,string"`
			ApplyToMarket       bool    `json:"applyToMarket"`
			Limit               int64   `json:"limit"`
			MaxNumAlgoOrders    int64   `json:"maxNumAlgoOrders"`
			MaxNumIcebergOrders int64   `json:"maxNumIcebergOrders"`
			MaxNumOrders        int64   `json:"maxNumOrders"`
		} `json:"filters"`
		Permissions []string `json:"permissions"`
	} `json:"symbols"`
}

// CoinInfo stores information about all supported coins
type CoinInfo struct {
	Coin              string  `json:"coin"`
	DepositAllEnable  bool    `json:"depositAllEnable"`
	WithdrawAllEnable bool    `json:"withdrawAllEnable"`
	Free              float64 `json:"free,string"`
	Freeze            float64 `json:"freeze,string"`
	IPOAble           float64 `json:"ipoable,string"`
	IPOing            float64 `json:"ipoing,string"`
	IsLegalMoney      bool    `json:"isLegalMoney"`
	Locked            float64 `json:"locked,string"`
	Name              string  `json:"name"`
	NetworkList       []struct {
		AddressRegex        string  `json:"addressRegex"`
		Coin                string  `json:"coin"`
		DepositDescription  string  `json:"depositDesc"` // shown only when "depositEnable" is false
		DepositEnable       bool    `json:"depositEnable"`
		IsDefault           bool    `json:"isDefault"`
		MemoRegex           string  `json:"memoRegex"`
		MinimumConfirmation uint16  `json:"minConfirm"`
		Name                string  `json:"name"`
		Network             string  `json:"network"`
		ResetAddressStatus  bool    `json:"resetAddressStatus"`
		SpecialTips         string  `json:"specialTips"`
		UnlockConfirm       uint16  `json:"unLockConfirm"`
		WithdrawDescription string  `json:"withdrawDesc"` // shown only when "withdrawEnable" is false
		WithdrawEnable      bool    `json:"withdrawEnable"`
		WithdrawFee         float64 `json:"withdrawFee,string"`
		WithdrawMinimum     float64 `json:"withdrawMin,string"`
		WithdrawMaximum     float64 `json:"withdrawMax,string"`
	} `json:"networkList"`
	Storage     float64 `json:"storage,string"`
	Trading     bool    `json:"trading"`
	Withdrawing float64 `json:"withdrawing,string"`
}

// OrderBookDataRequestParams represents Klines request data.
type OrderBookDataRequestParams struct {
	Symbol currency.Pair `json:"symbol"` // Required field; example LTCBTC,BTCUSDT
	Limit  int           `json:"limit"`  // Default 100; max 1000. Valid limits:[5, 10, 20, 50, 100, 500, 1000]
}

// OrderbookItem stores an individual orderbook item
type OrderbookItem struct {
	Price    float64
	Quantity float64
}

// OrderBookData is resp data from orderbook endpoint
type OrderBookData struct {
	Code         int         `json:"code"`
	Msg          string      `json:"msg"`
	LastUpdateID int64       `json:"lastUpdateId"`
	Bids         [][2]string `json:"bids"`
	Asks         [][2]string `json:"asks"`
}

// OrderBook actual structured data that can be used for orderbook
type OrderBook struct {
	Symbol       string
	LastUpdateID int64
	Code         int
	Msg          string
	Bids         []OrderbookItem
	Asks         []OrderbookItem
}

// DepthUpdateParams is used as an embedded type for WebsocketDepthStream
type DepthUpdateParams []struct {
	PriceLevel float64
	Quantity   float64
	ingnore    []interface{}
}

// WebsocketDepthStream is the difference for the update depth stream
type WebsocketDepthStream struct {
	Event         string           `json:"e"`
	Timestamp     time.Time        `json:"E"`
	Pair          string           `json:"s"`
	FirstUpdateID int64            `json:"U"`
	LastUpdateID  int64            `json:"u"`
	UpdateBids    [][2]interface{} `json:"b"`
	UpdateAsks    [][2]interface{} `json:"a"`
}

// RecentTradeRequestParams represents Klines request data.
type RecentTradeRequestParams struct {
	Symbol currency.Pair `json:"symbol"` // Required field. example LTCBTC, BTCUSDT
	Limit  int           `json:"limit"`  // Default 500; max 500.
}

// RecentTrade holds recent trade data
type RecentTrade struct {
	ID           int64     `json:"id"`
	Price        float64   `json:"price,string"`
	Quantity     float64   `json:"qty,string"`
	Time         time.Time `json:"time"`
	IsBuyerMaker bool      `json:"isBuyerMaker"`
	IsBestMatch  bool      `json:"isBestMatch"`
}

// TradeStream holds the trade stream data
type TradeStream struct {
	EventType      string    `json:"e"`
	EventTime      time.Time `json:"E"`
	Symbol         string    `json:"s"`
	TradeID        int64     `json:"t"`
	Price          string    `json:"p"`
	Quantity       string    `json:"q"`
	BuyerOrderID   int64     `json:"b"`
	SellerOrderID  int64     `json:"a"`
	TimeStamp      time.Time `json:"T"`
	Maker          bool      `json:"m"`
	BestMatchPrice bool      `json:"M"`
}

// KlineStream holds the kline stream data
type KlineStream struct {
	EventType string          `json:"e"`
	EventTime time.Time       `json:"E"`
	Symbol    string          `json:"s"`
	Kline     KlineStreamData `json:"k"`
}

// KlineStreamData defines kline streaming data
type KlineStreamData struct {
	StartTime                time.Time `json:"t"`
	CloseTime                time.Time `json:"T"`
	Symbol                   string    `json:"s"`
	Interval                 string    `json:"i"`
	FirstTradeID             int64     `json:"f"`
	LastTradeID              int64     `json:"L"`
	OpenPrice                float64   `json:"o,string"`
	ClosePrice               float64   `json:"c,string"`
	HighPrice                float64   `json:"h,string"`
	LowPrice                 float64   `json:"l,string"`
	Volume                   float64   `json:"v,string"`
	NumberOfTrades           int64     `json:"n"`
	KlineClosed              bool      `json:"x"`
	Quote                    float64   `json:"q,string"`
	TakerBuyBaseAssetVolume  float64   `json:"V,string"`
	TakerBuyQuoteAssetVolume float64   `json:"Q,string"`
}

// TickerStream holds the ticker stream data
type TickerStream struct {
	EventType              string    `json:"e"`
	EventTime              time.Time `json:"E"`
	Symbol                 string    `json:"s"`
	PriceChange            float64   `json:"p,string"`
	PriceChangePercent     float64   `json:"P,string"`
	WeightedAvgPrice       float64   `json:"w,string"`
	ClosePrice             float64   `json:"x,string"`
	LastPrice              float64   `json:"c,string"`
	LastPriceQuantity      float64   `json:"Q,string"`
	BestBidPrice           float64   `json:"b,string"`
	BestBidQuantity        float64   `json:"B,string"`
	BestAskPrice           float64   `json:"a,string"`
	BestAskQuantity        float64   `json:"A,string"`
	OpenPrice              float64   `json:"o,string"`
	HighPrice              float64   `json:"h,string"`
	LowPrice               float64   `json:"l,string"`
	TotalTradedVolume      float64   `json:"v,string"`
	TotalTradedQuoteVolume float64   `json:"q,string"`
	OpenTime               time.Time `json:"O"`
	CloseTime              time.Time `json:"C"`
	FirstTradeID           int64     `json:"F"`
	LastTradeID            int64     `json:"L"`
	NumberOfTrades         int64     `json:"n"`
}

// HistoricalTrade holds recent trade data
type HistoricalTrade struct {
	ID            int64     `json:"id"`
	Price         float64   `json:"price,string"`
	Quantity      float64   `json:"qty,string"`
	QuoteQuantity float64   `json:"quoteQty,string"`
	Time          time.Time `json:"time"`
	IsBuyerMaker  bool      `json:"isBuyerMaker"`
	IsBestMatch   bool      `json:"isBestMatch"`
}

// AggregatedTradeRequestParams holds request params
type AggregatedTradeRequestParams struct {
	Symbol currency.Pair // Required field; example LTCBTC, BTCUSDT
	// The first trade to retrieve
	FromID int64
	// The API seems to accept (start and end time) or FromID and no other combinations
	StartTime time.Time
	EndTime   time.Time
	// Default 500; max 1000.
	Limit int
}

// AggregatedTrade holds aggregated trade information
type AggregatedTrade struct {
	ATradeID       int64     `json:"a"`
	Price          float64   `json:"p,string"`
	Quantity       float64   `json:"q,string"`
	FirstTradeID   int64     `json:"f"`
	LastTradeID    int64     `json:"l"`
	TimeStamp      time.Time `json:"T"`
	Maker          bool      `json:"m"`
	BestMatchPrice bool      `json:"M"`
}

// IndexMarkPrice stores data for index and mark prices
type IndexMarkPrice struct {
	Symbol               string  `json:"symbol"`
	Pair                 string  `json:"pair"`
	MarkPrice            float64 `json:"markPrice,string"`
	IndexPrice           float64 `json:"indexPrice,string"`
	EstimatedSettlePrice float64 `json:"estimatedSettlePrice,string"`
	LastFundingRate      string  `json:"lastFundingRate"`
	NextFundingTime      int64   `json:"nextFundingTime"`
	Time                 int64   `json:"time"`
}

// CandleStick holds kline data
type CandleStick struct {
	OpenTime                 time.Time
	Open                     float64
	High                     float64
	Low                      float64
	Close                    float64
	Volume                   float64
	CloseTime                time.Time
	QuoteAssetVolume         float64
	TradeCount               float64
	TakerBuyAssetVolume      float64
	TakerBuyQuoteAssetVolume float64
}

// AveragePrice holds current average symbol price
type AveragePrice struct {
	Mins  int64   `json:"mins"`
	Price float64 `json:"price,string"`
}

// PriceChangeStats contains statistics for the last 24 hours trade
type PriceChangeStats struct {
	Symbol             string    `json:"symbol"`
	PriceChange        float64   `json:"priceChange,string"`
	PriceChangePercent float64   `json:"priceChangePercent,string"`
	WeightedAvgPrice   float64   `json:"weightedAvgPrice,string"`
	PrevClosePrice     float64   `json:"prevClosePrice,string"`
	LastPrice          float64   `json:"lastPrice,string"`
	LastQty            float64   `json:"lastQty,string"`
	BidPrice           float64   `json:"bidPrice,string"`
	AskPrice           float64   `json:"askPrice,string"`
	OpenPrice          float64   `json:"openPrice,string"`
	HighPrice          float64   `json:"highPrice,string"`
	LowPrice           float64   `json:"lowPrice,string"`
	Volume             float64   `json:"volume,string"`
	QuoteVolume        float64   `json:"quoteVolume,string"`
	OpenTime           time.Time `json:"openTime"`
	CloseTime          time.Time `json:"closeTime"`
	FirstID            int64     `json:"firstId"`
	LastID             int64     `json:"lastId"`
	Count              int64     `json:"count"`
}

// SymbolPrice holds basic symbol price
type SymbolPrice struct {
	Symbol string  `json:"symbol"`
	Price  float64 `json:"price,string"`
}

// BestPrice holds best price data
type BestPrice struct {
	Symbol   string  `json:"symbol"`
	BidPrice float64 `json:"bidPrice,string"`
	BidQty   float64 `json:"bidQty,string"`
	AskPrice float64 `json:"askPrice,string"`
	AskQty   float64 `json:"askQty,string"`
}

// NewOrderRequest request type
type NewOrderRequest struct {
	// Symbol (currency pair to trade)
	Symbol currency.Pair
	// Side Buy or Sell
	Side string
	// TradeType (market or limit order)
	TradeType RequestParamsOrderType
	// TimeInForce specifies how long the order remains in effect.
	// Examples are (Good Till Cancel (GTC), Immediate or Cancel (IOC) and Fill Or Kill (FOK))
	TimeInForce RequestParamsTimeForceType
	// Quantity is the total base qty spent or received in an order.
	Quantity float64
	// QuoteOrderQty is the total quote qty spent or received in a MARKET order.
	QuoteOrderQty    float64
	Price            float64
	NewClientOrderID string
	StopPrice        float64 // Used with STOP_LOSS, STOP_LOSS_LIMIT, TAKE_PROFIT, and TAKE_PROFIT_LIMIT orders.
	IcebergQty       float64 // Used with LIMIT, STOP_LOSS_LIMIT, and TAKE_PROFIT_LIMIT to create an iceberg order.
	NewOrderRespType string
}

// NewOrderResponse is the return structured response from the exchange
type NewOrderResponse struct {
	Code            int       `json:"code"`
	Msg             string    `json:"msg"`
	Symbol          string    `json:"symbol"`
	OrderID         int64     `json:"orderId"`
	ClientOrderID   string    `json:"clientOrderId"`
	TransactionTime time.Time `json:"transactTime"`
	Price           float64   `json:"price,string"`
	OrigQty         float64   `json:"origQty,string"`
	ExecutedQty     float64   `json:"executedQty,string"`
	// The cumulative amount of the quote that has been spent (with a BUY order) or received (with a SELL order).
	CumulativeQuoteQty float64 `json:"cummulativeQuoteQty,string"`
	Status             string  `json:"status"`
	TimeInForce        string  `json:"timeInForce"`
	Type               string  `json:"type"`
	Side               string  `json:"side"`
	Fills              []struct {
		Price           float64 `json:"price,string"`
		Qty             float64 `json:"qty,string"`
		Commission      float64 `json:"commission,string"`
		CommissionAsset string  `json:"commissionAsset"`
	} `json:"fills"`
}

// CancelOrderResponse is the return structured response from the exchange
type CancelOrderResponse struct {
	Symbol            string `json:"symbol"`
	OrigClientOrderID string `json:"origClientOrderId"`
	OrderID           int64  `json:"orderId"`
	ClientOrderID     string `json:"clientOrderId"`
}

// QueryOrderData holds query order data
type QueryOrderData struct {
	Code                int       `json:"code"`
	Msg                 string    `json:"msg"`
	Symbol              string    `json:"symbol"`
	OrderID             int64     `json:"orderId"`
	ClientOrderID       string    `json:"clientOrderId"`
	Price               float64   `json:"price,string"`
	OrigQty             float64   `json:"origQty,string"`
	ExecutedQty         float64   `json:"executedQty,string"`
	Status              string    `json:"status"`
	TimeInForce         string    `json:"timeInForce"`
	Type                string    `json:"type"`
	Side                string    `json:"side"`
	StopPrice           float64   `json:"stopPrice,string"`
	IcebergQty          float64   `json:"icebergQty,string"`
	Time                time.Time `json:"time"`
	IsWorking           bool      `json:"isWorking"`
	CummulativeQuoteQty float64   `json:"cummulativeQuoteQty,string"`
	OrderListID         int64     `json:"orderListId"`
	OrigQuoteOrderQty   float64   `json:"origQuoteOrderQty,string"`
	UpdateTime          time.Time `json:"updateTime"`
}

// Balance holds query order data
type Balance struct {
	Asset  string `json:"asset"`
	Free   string `json:"free"`
	Locked string `json:"locked"`
}

// Account holds the account data
type Account struct {
	MakerCommission  int       `json:"makerCommission"`
	TakerCommission  int       `json:"takerCommission"`
	BuyerCommission  int       `json:"buyerCommission"`
	SellerCommission int       `json:"sellerCommission"`
	CanTrade         bool      `json:"canTrade"`
	CanWithdraw      bool      `json:"canWithdraw"`
	CanDeposit       bool      `json:"canDeposit"`
	UpdateTime       time.Time `json:"updateTime"`
	Balances         []Balance `json:"balances"`
}

// MarginAccount holds the margin account data
type MarginAccount struct {
	BorrowEnabled       bool                 `json:"borrowEnabled"`
	MarginLevel         float64              `json:"marginLevel,string"`
	TotalAssetOfBtc     float64              `json:"totalAssetOfBtc,string"`
	TotalLiabilityOfBtc float64              `json:"totalLiabilityOfBtc,string"`
	TotalNetAssetOfBtc  float64              `json:"totalNetAssetOfBtc,string"`
	TradeEnabled        bool                 `json:"tradeEnabled"`
	TransferEnabled     bool                 `json:"transferEnabled"`
	UserAssets          []MarginAccountAsset `json:"userAssets"`
}

// MarginAccountAsset holds each individual margin account asset
type MarginAccountAsset struct {
	Asset    string  `json:"asset"`
	Borrowed float64 `json:"borrowed,string"`
	Free     float64 `json:"free,string"`
	Interest float64 `json:"interest,string"`
	Locked   float64 `json:"locked,string"`
	NetAsset float64 `json:"netAsset,string"`
}

// RequestParamsTimeForceType Time in force
type RequestParamsTimeForceType string

var (
	// BinanceRequestParamsTimeGTC GTC
	BinanceRequestParamsTimeGTC = RequestParamsTimeForceType("GTC")

	// BinanceRequestParamsTimeIOC IOC
	BinanceRequestParamsTimeIOC = RequestParamsTimeForceType("IOC")

	// BinanceRequestParamsTimeFOK FOK
	BinanceRequestParamsTimeFOK = RequestParamsTimeForceType("FOK")
)

// RequestParamsOrderType trade order type
type RequestParamsOrderType string

var (
	// BinanceRequestParamsOrderLimit Limit order
	BinanceRequestParamsOrderLimit = RequestParamsOrderType("LIMIT")

	// BinanceRequestParamsOrderMarket Market order
	BinanceRequestParamsOrderMarket = RequestParamsOrderType("MARKET")

	// BinanceRequestParamsOrderStopLoss STOP_LOSS
	BinanceRequestParamsOrderStopLoss = RequestParamsOrderType("STOP_LOSS")

	// BinanceRequestParamsOrderStopLossLimit STOP_LOSS_LIMIT
	BinanceRequestParamsOrderStopLossLimit = RequestParamsOrderType("STOP_LOSS_LIMIT")

	// BinanceRequestParamsOrderTakeProfit TAKE_PROFIT
	BinanceRequestParamsOrderTakeProfit = RequestParamsOrderType("TAKE_PROFIT")

	// BinanceRequestParamsOrderTakeProfitLimit TAKE_PROFIT_LIMIT
	BinanceRequestParamsOrderTakeProfitLimit = RequestParamsOrderType("TAKE_PROFIT_LIMIT")

	// BinanceRequestParamsOrderLimitMarker LIMIT_MAKER
	BinanceRequestParamsOrderLimitMarker = RequestParamsOrderType("LIMIT_MAKER")
)

// KlinesRequestParams represents Klines request data.
type KlinesRequestParams struct {
	Symbol    currency.Pair // Required field; example LTCBTC, BTCUSDT
	Interval  string        // Time interval period
	Limit     int           // Default 500; max 500.
	StartTime time.Time
	EndTime   time.Time
}

// DepositHistory stores deposit history info
type DepositHistory struct {
	Amount        float64 `json:"amount,string"`
	Coin          string  `json:"coin"`
	Network       string  `json:"network"`
	Status        uint8   `json:"status"`
	Address       string  `json:"address"`
	AddressTag    string  `json:"adressTag"`
	TransactionID string  `json:"txId"`
	InsertTime    float64 `json:"insertTime"`
	TransferType  uint8   `json:"transferType"`
	ConfirmTimes  string  `json:"confirmTimes"`
}

// WithdrawResponse contains status of withdrawal request
type WithdrawResponse struct {
	ID string `json:"id"`
}

// WithdrawStatusResponse defines a withdrawal status response
type WithdrawStatusResponse struct {
	Address         string  `json:"address"`
	Amount          float64 `json:"amount,string"`
	ApplyTime       string  `json:"applyTime"`
	Coin            string  `json:"coin"`
	ID              string  `json:"id"`
	WithdrawOrderID string  `json:"withdrawOrderId"`
	Network         string  `json:"network"`
	TransferType    uint8   `json:"transferType"`
	Status          int64   `json:"status"`
	TransactionFee  float64 `json:"transactionFee,string"`
	TransactionID   string  `json:"txId"`
	ConfirmNumber   int64   `json:"confirmNo"`
}

// DepositAddress stores the deposit address info
type DepositAddress struct {
	Address string `json:"address"`
	Coin    string `json:"coin"`
	Tag     string `json:"tag"`
	URL     string `json:"url"`
}

// UserAccountStream contains a key to maintain an authorised
// websocket connection
type UserAccountStream struct {
	ListenKey string `json:"listenKey"`
}

type wsAccountInfo struct {
	Stream string            `json:"stream"`
	Data   WsAccountInfoData `json:"data"`
}

// WsAccountInfoData defines websocket account info data
type WsAccountInfoData struct {
	CanDeposit       bool      `json:"D"`
	CanTrade         bool      `json:"T"`
	CanWithdraw      bool      `json:"W"`
	EventTime        time.Time `json:"E"`
	LastUpdated      time.Time `json:"u"`
	BuyerCommission  float64   `json:"b"`
	MakerCommission  float64   `json:"m"`
	SellerCommission float64   `json:"s"`
	TakerCommission  float64   `json:"t"`
	EventType        string    `json:"e"`
	Currencies       []struct {
		Asset     string  `json:"a"`
		Available float64 `json:"f,string"`
		Locked    float64 `json:"l,string"`
	} `json:"B"`
}

type wsAccountPosition struct {
	Stream string                `json:"stream"`
	Data   WsAccountPositionData `json:"data"`
}

// WsAccountPositionData defines websocket account position data
type WsAccountPositionData struct {
	Currencies []struct {
		Asset     string  `json:"a"`
		Available float64 `json:"f,string"`
		Locked    float64 `json:"l,string"`
	} `json:"B"`
	EventTime   time.Time `json:"E"`
	LastUpdated time.Time `json:"u"`
	EventType   string    `json:"e"`
}

type wsBalanceUpdate struct {
	Stream string              `json:"stream"`
	Data   WsBalanceUpdateData `json:"data"`
}

// WsBalanceUpdateData defines websocket account balance data
type WsBalanceUpdateData struct {
	EventTime    time.Time `json:"E"`
	ClearTime    time.Time `json:"T"`
	BalanceDelta float64   `json:"d,string"`
	Asset        string    `json:"a"`
	EventType    string    `json:"e"`
}

type wsOrderUpdate struct {
	Stream string            `json:"stream"`
	Data   WsOrderUpdateData `json:"data"`
}

// WsOrderUpdateData defines websocket account order update data
type WsOrderUpdateData struct {
	EventType                         string    `json:"e"`
	EventTime                         time.Time `json:"E"`
	Symbol                            string    `json:"s"`
	ClientOrderID                     string    `json:"c"`
	Side                              string    `json:"S"`
	OrderType                         string    `json:"o"`
	TimeInForce                       string    `json:"f"`
	Quantity                          float64   `json:"q,string"`
	Price                             float64   `json:"p,string"`
	StopPrice                         float64   `json:"P,string"`
	IcebergQuantity                   float64   `json:"F,string"`
	OrderListID                       int64     `json:"g"`
	CancelledClientOrderID            string    `json:"C"`
	CurrentExecutionType              string    `json:"x"`
	OrderStatus                       string    `json:"X"`
	RejectionReason                   string    `json:"r"`
	OrderID                           int64     `json:"i"`
	LastExecutedQuantity              float64   `json:"l,string"`
	CumulativeFilledQuantity          float64   `json:"z,string"`
	LastExecutedPrice                 float64   `json:"L,string"`
	Commission                        float64   `json:"n,string"`
	CommissionAsset                   string    `json:"N"`
	TransactionTime                   time.Time `json:"T"`
	TradeID                           int64     `json:"t"`
	Ignored                           int64     `json:"I"` // Must be ignored explicitly, otherwise it overwrites 'i'.
	IsOnOrderBook                     bool      `json:"w"`
	IsMaker                           bool      `json:"m"`
	Ignored2                          bool      `json:"M"` // See the comment for "I".
	OrderCreationTime                 time.Time `json:"O"`
	CumulativeQuoteTransactedQuantity float64   `json:"Z,string"`
	LastQuoteAssetTransactedQuantity  float64   `json:"Y,string"`
	QuoteOrderQuantity                float64   `json:"Q,string"`
}

type wsListStatus struct {
	Stream string           `json:"stream"`
	Data   WsListStatusData `json:"data"`
}

// WsListStatusData defines websocket account listing status data
type WsListStatusData struct {
	ListClientOrderID string    `json:"C"`
	EventTime         time.Time `json:"E"`
	ListOrderStatus   string    `json:"L"`
	Orders            []struct {
		ClientOrderID string `json:"c"`
		OrderID       int64  `json:"i"`
		Symbol        string `json:"s"`
	} `json:"O"`
	TransactionTime time.Time `json:"T"`
	ContingencyType string    `json:"c"`
	EventType       string    `json:"e"`
	OrderListID     int64     `json:"g"`
	ListStatusType  string    `json:"l"`
	RejectionReason string    `json:"r"`
	Symbol          string    `json:"s"`
}

// WsPayload defines the payload through the websocket connection
type WsPayload struct {
	Method string   `json:"method"`
	Params []string `json:"params"`
	ID     int64    `json:"id"`
}

// CrossMarginInterestData stores cross margin data for borrowing
type CrossMarginInterestData struct {
	Code          int64  `json:"code,string"`
	Message       string `json:"message"`
	MessageDetail string `json:"messageDetail"`
	Data          []struct {
		AssetName string `json:"assetName"`
		Specs     []struct {
			VipLevel          string `json:"vipLevel"`
			DailyInterestRate string `json:"dailyInterestRate"`
			BorrowLimit       string `json:"borrowLimit"`
		} `json:"specs"`
	} `json:"data"`
	Success bool `json:"success"`
}

// orderbookManager defines a way of managing and maintaining synchronisation
// across connections and assets.
type orderbookManager struct {
	state map[currency.Code]map[currency.Code]map[asset.Item]*update
	sync.Mutex

	jobs chan job
}

type update struct {
	buffer            chan *WebsocketDepthStream
	fetchingBook      bool
	initialSync       bool
	needsFetchingBook bool
	lastUpdateID      int64
}

// job defines a synchonisation job that tells a go routine to fetch an
// orderbook via the REST protocol
type job struct {
	Pair currency.Pair
}

// AllCoinsInfo defines extended coin information associated with an account.
type AllCoinsInfo struct {
	Coin             currency.Code `json:"coin"`
	DepositAllEnable bool          `json:"depositAllEnable"`
	Free             float64       `json:"free,string"`
	Freeze           float64       `json:"freeze,string"`
	Ipoable          float64       `json:"ipoable,string"`
	Ipoing           float64       `json:"ipoing,string"`
	IsLegalMoney     bool          `json:"isLegalMoney"`
	Locked           float64       `json:"locked,string"`
	Name             string        `json:"name"`
	NetworkList      []struct {
		AddressRegex            string        `json:"addressRegex"`
		Coin                    currency.Code `json:"coin"`
		DepositDesc             string        `json:"depositDesc,omitempty"`
		DepositEnable           bool          `json:"depositEnable"`
		IsDefault               bool          `json:"isDefault"`
		MemoRegex               string        `json:"memoRegex"`
		MinConfirm              int           `json:"minConfirm"`
		Name                    string        `json:"name"`
		Network                 currency.Code `json:"network"`
		ResetAddressStatus      bool          `json:"resetAddressStatus"`
		SpecialTips             string        `json:"specialTips"`
		UnLockConfirm           int           `json:"unLockConfirm"`
		WithdrawDesc            string        `json:"withdrawDesc,omitempty"`
		WithdrawEnable          bool          `json:"withdrawEnable"`
		WithdrawFee             float64       `json:"withdrawFee,string"`
		WithdrawIntegerMultiple float64       `json:"withdrawIntegerMultiple,string"`
		WithdrawMax             float64       `json:"withdrawMax,string"`
		WithdrawMin             float64       `json:"withdrawMin,string"`
		SameAddress             bool          `json:"sameAddress"`
	} `json:"networkList"`
	Storage           float64 `json:"storage,string"`
	Trading           bool    `json:"trading"`
	WithdrawAllEnable bool    `json:"withdrawAllEnable"`
	Withdrawing       float64 `json:"withdrawing,string"`
}

// bankTransferFees defines current bank transfer fees, subject to change.
var bankTransferFees = []fee.Transfer{
	{BankTransfer: bank.WireTransfer,
		Currency:   currency.AUD,
		Withdrawal: fee.Convert(0)},
	{BankTransfer: bank.WireTransfer,
		Currency:   currency.BRL,
		Deposit:    fee.Convert(0),
		Withdrawal: fee.Convert(2.6)},
	{BankTransfer: bank.WireTransfer,
		Currency:   currency.PHP,
		Deposit:    fee.Convert(25),
		Withdrawal: fee.Convert(60)},
	{BankTransfer: bank.WireTransfer,
		Currency:   currency.TRY,
		Deposit:    fee.Convert(0),
		Withdrawal: fee.Convert(0)},
	{BankTransfer: bank.WireTransfer,
		Currency:   currency.UGX,
		Withdrawal: fee.Convert(8000)},

	{BankTransfer: bank.PayIDOsko,
		Currency: currency.AUD,
		Deposit:  fee.Convert(0)},

	{BankTransfer: bank.BankCardVisa,
		Currency:   currency.EUR,
		Withdrawal: fee.Convert(0.01), IsPercentage: true},
	{BankTransfer: bank.BankCardVisa,
		Currency:   currency.GBP,
		Withdrawal: fee.Convert(0.01), IsPercentage: true},

	{BankTransfer: bank.BankCardMastercard,
		Currency: currency.EUR,
		Deposit:  fee.Convert(0.018), IsPercentage: true},
	{BankTransfer: bank.BankCardMastercard,
		Currency:   currency.GBP,
		Withdrawal: fee.Convert(0.01), IsPercentage: true},
	{BankTransfer: bank.BankCardMastercard,
		Currency: currency.HKD,
		Deposit:  fee.Convert(0.035), IsPercentage: true},
	{BankTransfer: bank.BankCardMastercard,
		Currency: currency.PEN,
		Deposit:  fee.Convert(0.035), IsPercentage: true},

	{BankTransfer: bank.CreditCardMastercard,
		Currency:   currency.RUB,
		Withdrawal: fee.Convert(250)},

	{BankTransfer: bank.Sofort,
		Currency: currency.EUR,
		Deposit:  fee.Convert(0.02), IsPercentage: true},

	{BankTransfer: bank.SEPA,
		Currency: currency.EUR,
		Deposit:  fee.Convert(0), Withdrawal: fee.Convert(1.5)},

	{BankTransfer: bank.P2P,
		Currency: currency.EUR,
		Deposit:  fee.Convert(0)},

	{BankTransfer: bank.AdvCash,
		Currency: currency.EUR,
		Deposit:  fee.Convert(0), Withdrawal: fee.Convert(0)},
	{BankTransfer: bank.AdvCash,
		Currency: currency.KZT,
		Deposit:  fee.Convert(0), Withdrawal: fee.Convert(0)},
	{BankTransfer: bank.AdvCash,
		Currency:   currency.RUB,
		Withdrawal: fee.Convert(0)},
	{BankTransfer: bank.AdvCash,
		Currency:     currency.TRY,
		Deposit:      fee.Convert(0.03),
		Withdrawal:   fee.Convert(0),
		IsPercentage: true},
	{BankTransfer: bank.AdvCash,
		Currency:     currency.UAH,
		Deposit:      fee.Convert(0.005),
		Withdrawal:   fee.Convert(0),
		IsPercentage: true},

	{BankTransfer: bank.Etana,
		Currency:     currency.EUR,
		Deposit:      fee.Convert(0.001),
		Withdrawal:   fee.Convert(0.001),
		IsPercentage: true},

	{BankTransfer: bank.FasterPaymentService,
		Currency: currency.GBP,
		Deposit:  fee.Convert(0.05), Withdrawal: fee.Convert(0.5)},

	{BankTransfer: bank.MobileMoney,
		Currency: currency.GHS,
		Deposit:  fee.Convert(0.025), IsPercentage: true},
	{BankTransfer: bank.MobileMoney,
		Currency:     currency.UGX,
		Deposit:      fee.Convert(0.035),
		Withdrawal:   fee.Convert(0.015),
		IsPercentage: true},

	{BankTransfer: bank.CashTransfer,
		Currency:   currency.NGN,
		Withdrawal: fee.Convert(0)},

	{BankTransfer: bank.YandexMoney,
		Currency:   currency.RUB,
		Withdrawal: fee.Convert(0.025), IsPercentage: true},

	{BankTransfer: bank.BankCardMIR,
		Currency:   currency.RUB,
		Withdrawal: fee.Convert(0.021), IsPercentage: true},

	{BankTransfer: bank.Payeer,
		Currency:     currency.RUB,
		Deposit:      fee.Convert(0),
		Withdrawal:   fee.Convert(0.01),
		IsPercentage: true},

	{BankTransfer: bank.GEOPay,
		Currency:     currency.UAH,
		Deposit:      fee.Convert(0.008),
		Withdrawal:   fee.Convert(0),
		IsPercentage: true},

	{BankTransfer: bank.SettlePay,
		Currency:     currency.UAH,
		Deposit:      fee.Convert(0.008),
		Withdrawal:   fee.Convert(0),
		IsPercentage: true},

	{BankTransfer: bank.ExchangeFiatDWChannelSignetUSD,
		Currency: currency.USD,
		Deposit:  fee.Convert(0), Withdrawal: fee.Convert(0)},

	{BankTransfer: bank.ExchangeFiatDWChannelSwiftSignatureBar,
		Currency: currency.USD,
		Deposit:  fee.Convert(0), Withdrawal: fee.Convert(15)},
}

// Tier defines maker and taker fees for a fee tier
type Tier struct {
	Maker float64
	Taker float64
}

var coinMarginedFeeTier = map[int64]Tier{
	0: {Maker: 0.0001, Taker: 0.0005},
	1: {Maker: 0.00008, Taker: 0.00045},
	2: {Maker: 0.00005, Taker: 0.0004},
	3: {Maker: 0.00003, Taker: 0.0003},
	4: {Maker: 0, Taker: 0.00025},
	5: {Maker: -0.00005, Taker: 0.00024},
	6: {Maker: -0.00006, Taker: 0.00024},
	7: {Maker: -0.00007, Taker: 0.00024},
	8: {Maker: -0.00008, Taker: 0.00024},
	9: {Maker: -0.00009, Taker: 0.00024},
}

var usdMarginedFeeTier = map[int64]Tier{
	0: {Maker: 0.0002, Taker: 0.0004},
	1: {Maker: 0.00016, Taker: 0.0004},
	2: {Maker: 0.00014, Taker: 0.00035},
	3: {Maker: 0.00012, Taker: 0.00032},
	4: {Maker: 0.0001, Taker: 0.0003},
	5: {Maker: 0.00008, Taker: 0.00027},
	6: {Maker: 0.00006, Taker: 0.00025},
	7: {Maker: 0.00004, Taker: 0.00022},
	8: {Maker: 0.00002, Taker: 0.0002},
	9: {Maker: 0, Taker: 0.00017},
}

var transferFees = []fee.Transfer{
	{Currency: currency.ONEINCH, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.074, 0.15)},
	{Currency: currency.ONEINCH, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(8.57, 17)},
	{Currency: currency.AGLD, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(12, 24)},
	{Currency: currency.AUDIO, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(13, 26)},
	{Currency: currency.AION, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.1, 0.2)},
	{Currency: currency.AION, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(204, 408)},
	{Currency: currency.AR, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.03, 0.1)},
	{Currency: currency.ACA, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0, 0)},
	{Currency: currency.ARDR, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(2, 4)},
	{Currency: currency.ACM, Chain: "CHZ", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0, 1)},
	{Currency: currency.ADA, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1, 10)},
	{Currency: currency.ADA, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.18, 0.36)},
	{Currency: currency.ADA, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.18, 0.36)},
	{Currency: currency.ADD, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(100, 200)},
	{Currency: currency.ADX, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.41, 0.82)},
	{Currency: currency.ADX, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(47, 94)},
	{Currency: currency.AGI, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(83, 166)},
	{Currency: currency.ATOM, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.005, 0.01)},
	{Currency: currency.ATOM, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.0099, 0.02)},
	{Currency: currency.ATOM, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.0099, 0.02)},
	{Currency: currency.AUCTION, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.0098, 0.02)},
	{Currency: currency.AUCTION, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1.14, 2.28)},
	{Currency: currency.AMB, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1, 2)},
	{Currency: currency.AMB, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(887, 1774)},
	{Currency: currency.AERGO, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.84, 1.68)},
	{Currency: currency.AERGO, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(96, 192)},
	{Currency: currency.AMP, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(571, 1142)},
	{Currency: currency.AUTO, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.00031, 0.00062)},
	{Currency: currency.ANTOLD, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1, 2)},
	{Currency: currency.ANT, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(6.65, 13)},
	{Currency: currency.ALICE, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.012, 0.024)},
	{Currency: currency.ALICE, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1.34, 2.68)},
	{Currency: currency.ARPA, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1.93, 3.86)},
	{Currency: currency.ARPA, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1.93, 3.86)},
	{Currency: currency.ARPA, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(222, 444)},
	{Currency: currency.ASTR, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0, 0)},
	{Currency: currency.ARK, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.2, 0.4)},
	{Currency: currency.ARN, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.26, 0.52)},
	{Currency: currency.ARN, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(17, 34)},
	{Currency: currency.ANKR, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(2.51, 5.02)},
	{Currency: currency.ANKR, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(2.5, 5)},
	{Currency: currency.ANKR, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(288, 576)},
	{Currency: currency.ALGO, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.01, 10)},
	{Currency: currency.ASR, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.02, 0.04)},
	{Currency: currency.ASR, Chain: "CHZ", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0, 1)},
	{Currency: currency.AXSOLD, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0, 15)},
	{Currency: currency.AST, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(55, 110)},
	{Currency: currency.ATA, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.29, 0.58)},
	{Currency: currency.ATA, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(33, 66)},
	{Currency: currency.ATD, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(100, 200)},
	{Currency: currency.ATM, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.02, 0.04)},
	{Currency: currency.ATM, Chain: "CHZ", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0, 1)},
	{Currency: currency.APPC, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(415, 830)},
	{Currency: currency.ALPHA, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.33, 0.66)},
	{Currency: currency.ALPHA, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(38, 76)},
	{Currency: currency.AVA, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.11, 0.22)},
	{Currency: currency.AVA, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.11, 0.22)},
	{Currency: currency.AVA, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(13, 26)},
	{Currency: currency.AAVE, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.0012, 0.0024)},
	{Currency: currency.AAVE, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.0012, 0.0024)},
	{Currency: currency.AAVE, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.27, 0.54)},
	{Currency: currency.AXS, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.0021, 0.0042)},
	{Currency: currency.AXS, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.24, 0.48)},
	{Currency: currency.AXS, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.005, 0.2)},
	{Currency: currency.AVAX, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.01, 0.1)},
	{Currency: currency.AVAX, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.01, 0.4)},
	{Currency: currency.AVAX, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.0025, 0.005)},
	{Currency: currency.ADXOLD, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(5, 10)},
	{Currency: currency.ALPACA, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.42, 0.84)},
	{Currency: currency.AGIX, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(160, 320)},
	{Currency: currency.AKRO, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1081, 2162)},
	{Currency: currency.BCD, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.01, 0.02)},
	{Currency: currency.BCH, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.001, 0.002)},
	{Currency: currency.BCH, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.00048, 0.00096)},
	{Currency: currency.BCH, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.00048, 0.00096)},
	{Currency: currency.BCH, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.055, 0.11)},
	{Currency: currency.BCX, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.5, 0.5)},
	{Currency: currency.BKRW, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0, 20)},
	{Currency: currency.BEL, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.13, 0.26)},
	{Currency: currency.BEL, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.13, 0.26)},
	{Currency: currency.BEL, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(14, 28)},
	{Currency: currency.BZRX, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1.05, 2.1)},
	{Currency: currency.BZRX, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(120, 240)},
	{Currency: currency.BCHA, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.0036, 0.0072)},
	{Currency: currency.BCHA, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.0036, 0.0072)},
	{Currency: currency.BLINK, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(50, 100)},
	{Currency: currency.BLINK, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(50, 100)},
	{Currency: currency.BURGER, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.078, 0.16)},
	{Currency: currency.BURGER, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.078, 0.16)},
	{Currency: currency.BULL, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0, 0.0000001)},
	{Currency: currency.BVND, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(6543, 13086)},
	{Currency: currency.BLZ, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.76, 1.52)},
	{Currency: currency.BLZ, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(87, 174)},
	{Currency: currency.BOBA, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.33, 0.66)},
	{Currency: currency.BNC, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0, 0)},
	{Currency: currency.BNB, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.0005, 0.01)},
	{Currency: currency.BNB, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.0005, 0.02)},
	{Currency: currency.BNB, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0, 0.012)},
	{Currency: currency.BADGER, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1.3, 2.6)},
	{Currency: currency.BNT, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.074, 0.15)},
	{Currency: currency.BNT, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(8.56, 17)},
	{Currency: currency.BNX, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.0016, 0.0032)},
	{Currency: currency.BOT, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.0001, 0.0002)},
	{Currency: currency.BOT, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.012, 0.024)},
	{Currency: currency.BAKE, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.16, 0.32)},
	{Currency: currency.BAKE, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.16, 0.32)},
	{Currency: currency.BETH, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.000075, 0.00015)},
	{Currency: currency.BETH, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.0087, 0.017)},
	{Currency: currency.BQX, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0, 1)},
	{Currency: currency.BETA, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.27, 0.54)},
	{Currency: currency.BETA, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(30, 60)},
	{Currency: currency.BRD, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(22, 44)},
	{Currency: currency.BUSD, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.5, 10)},
	{Currency: currency.BUSD, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0, 10)},
	{Currency: currency.BUSD, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(25, 50)},
	{Currency: currency.BNBBEAR, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.011, 0.022)},
	{Currency: currency.BAND, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.01, 0.02)},
	{Currency: currency.BAND, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.041, 0.082)},
	{Currency: currency.BAND, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.041, 0.082)},
	{Currency: currency.BAND, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(4.77, 9.54)},
	{Currency: currency.BTC, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.0000051, 0.00001)},
	{Currency: currency.BTC, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.0000051, 0.00001)},
	{Currency: currency.BTC, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.0005, 0.001)},
	{Currency: currency.BTC, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.00059, 0.0012)},
	{Currency: currency.BTC, Chain: "SegWit", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.0005, 0.001)},
	{Currency: currency.BTG, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.0052, 0.01)},
	{Currency: currency.BTG, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.001, 0.002)},
	{Currency: currency.BTM, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(5, 10)},
	{Currency: currency.BTS, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1, 2)},
	{Currency: currency.BTT, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(93, 186)},
	{Currency: currency.BTT, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(93, 186)},
	{Currency: currency.BTT, Chain: "TRC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(30, 60)},
	{Currency: currency.BOLT, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(10, 20)},
	{Currency: currency.BOLT, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(10, 20)},
	{Currency: currency.BOND, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.4, 0.8)},
	{Currency: currency.BIDR, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(4282, 8564)},
	{Currency: currency.BIDR, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(4281, 8562)},
	{Currency: currency.BCHSV, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.01, 0.02)},
	{Currency: currency.BGBP, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(2.81, 5.62)},
	{Currency: currency.BIFI, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.00024, 0.00048)},
	{Currency: currency.BIFI, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.00024, 0.00048)},
	{Currency: currency.BTCST, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.0091, 0.018)},
	{Currency: currency.BTCST, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.0091, 0.018)},
	{Currency: currency.BEAR, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0, 0.00001)},
	{Currency: currency.BEAM, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.1, 0.2)},
	{Currency: currency.BNBBULL, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.012, 0.024)},
	{Currency: currency.BAL, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.014, 0.028)},
	{Currency: currency.BAL, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1.63, 3.26)},
	{Currency: currency.BAR, Chain: "CHZ", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0, 1)},
	{Currency: currency.BAT, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.27, 0.54)},
	{Currency: currency.BAT, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.27, 0.54)},
	{Currency: currency.BAT, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(30, 60)},
	{Currency: currency.CDT, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(201, 402)},
	{Currency: currency.COVER, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.005, 0.01)},
	{Currency: currency.CFX, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1.04, 2.08)},
	{Currency: currency.CHR, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.24, 0.48)},
	{Currency: currency.CHR, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(28, 56)},
	{Currency: currency.CHZ, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.59, 1.18)},
	{Currency: currency.CHZ, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(68, 136)},
	{Currency: currency.CKB, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1, 100)},
	{Currency: currency.CLV, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.27, 0.54)},
	{Currency: currency.CLV, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(31, 62)},
	{Currency: currency.CHESS, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.12, 0.24)},
	{Currency: currency.CND, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(2123, 4246)},
	{Currency: currency.COS, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(13, 26)},
	{Currency: currency.COS, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(13, 26)},
	{Currency: currency.COS, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1564, 3128)},
	{Currency: currency.CITY, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.024, 0.048)},
	{Currency: currency.CITY, Chain: "CHZ", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0, 1)},
	{Currency: currency.CELR, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(2.39, 4.78)},
	{Currency: currency.CELR, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(276, 552)},
	{Currency: currency.CELO, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.001, 0.002)},
	{Currency: currency.CRV, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(6.13, 12)},
	{Currency: currency.CTK, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.17, 0.34)},
	{Currency: currency.CTK, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.17, 0.34)},
	{Currency: currency.CTK, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.01, 0.02)},
	{Currency: currency.CTR, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(35, 70)},
	{Currency: currency.CAKE, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.021, 0.042)},
	{Currency: currency.CAKE, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.021, 0.042)},
	{Currency: currency.CVC, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(69, 138)},
	{Currency: currency.CVP, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.2, 0.4)},
	{Currency: currency.CVP, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(23, 46)},
	{Currency: currency.CTSI, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.3, 0.6)},
	{Currency: currency.CTSI, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(35, 70)},
	{Currency: currency.C98, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.093, 0.19)},
	{Currency: currency.C98, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(10, 20)},
	{Currency: currency.COCOS, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.06, 0.12)},
	{Currency: currency.COCOS, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.06, 0.12)},
	{Currency: currency.COCOS, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(6.99, 13)},
	{Currency: currency.COVEROLD, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.055, 0.11)},
	{Currency: currency.CTXC, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.1, 0.2)},
	{Currency: currency.CTXC, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(151, 302)},
	{Currency: currency.COMP, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.0011, 0.0022)},
	{Currency: currency.COMP, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.0011, 0.0022)},
	{Currency: currency.COMP, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.12, 0.24)},
	{Currency: currency.CHAT, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0, 200)},
	{Currency: currency.CAN, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(5, 10)},
	{Currency: currency.CAN, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(5, 10)},
	{Currency: currency.CREAM, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.007, 0.014)},
	{Currency: currency.CREAM, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.81, 1.62)},
	{Currency: currency.CBK, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1.2, 2.4)},
	{Currency: currency.CBM, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(200, 400)},
	{Currency: currency.CBM, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(150000, 300000)},
	{Currency: currency.COTI, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.71, 1.42)},
	{Currency: currency.COTI, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.71, 1.42)},
	{Currency: currency.COTI, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(82, 164)},
	{Currency: currency.DGB, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.2, 0.4)},
	{Currency: currency.DGD, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.026, 0.052)},
	{Currency: currency.DIA, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.17, 0.34)},
	{Currency: currency.DIA, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(19, 38)},
	{Currency: currency.DF, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1.74, 3.48)},
	{Currency: currency.DF, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(200, 400)},
	{Currency: currency.DREPOLD, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(661, 1322)},
	{Currency: currency.DLT, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(836, 1672)},
	{Currency: currency.DNT, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(675, 1350)},
	{Currency: currency.DON, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.0088, 0.018)},
	{Currency: currency.DOT, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.0076, 0.015)},
	{Currency: currency.DOT, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.0076, 0.015)},
	{Currency: currency.DOT, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.1, 1.5)},
	{Currency: currency.DOT, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.88, 1.76)},
	{Currency: currency.DEGO, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.024, 0.048)},
	{Currency: currency.DEGO, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(2.72, 5.44)},
	{Currency: currency.DREP, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.26, 0.52)},
	{Currency: currency.DREP, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(29, 58)},
	{Currency: currency.DOCK, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(5, 10)},
	{Currency: currency.DENT, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(6398, 12796)},
	{Currency: currency.DODO, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.22, 0.44)},
	{Currency: currency.DODO, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(25, 50)},
	{Currency: currency.DOGE, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1.36, 2.72)},
	{Currency: currency.DOGE, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1.36, 2.72)},
	{Currency: currency.DOGE, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0, 10)},
	{Currency: currency.DUSK, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.93, 1.86)},
	{Currency: currency.DUSK, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.93, 1.86)},
	{Currency: currency.DUSK, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(107, 214)},
	{Currency: currency.DEXE, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.023, 0.046)},
	{Currency: currency.DEXE, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(2.61, 5.22)},
	{Currency: currency.DAI, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1, 10)},
	{Currency: currency.DAI, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.8, 10)},
	{Currency: currency.DAI, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(15, 30)},
	{Currency: currency.DAR, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.071, 0.14)},
	{Currency: currency.DAR, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(8.16, 16)},
	{Currency: currency.DASH, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.002, 0.004)},
	{Currency: currency.DCR, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.01, 0.02)},
	{Currency: currency.DATA, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(2.23, 4.46)},
	{Currency: currency.DATA, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(257, 514)},
	{Currency: currency.DYDX, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(8.62, 17)},
	{Currency: currency.EASY, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.018, 0.036)},
	{Currency: currency.EASY, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(2.02, 4.04)},
	{Currency: currency.ELF, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.6, 1.2)},
	{Currency: currency.ELF, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.6, 1.2)},
	{Currency: currency.ELF, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(68, 136)},
	{Currency: currency.EZ, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.072, 0.14)},
	{Currency: currency.EZ, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(8.38, 16)},
	{Currency: currency.ENG, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0, 1)},
	{Currency: currency.ENJ, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(8.14, 16)},
	{Currency: currency.ENS, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.7, 1.4)},
	{Currency: currency.EON, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(10, 20)},
	{Currency: currency.EOP, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(5, 10)},
	{Currency: currency.EOS, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.072, 0.14)},
	{Currency: currency.EOS, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.072, 0.14)},
	{Currency: currency.EOS, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.1, 0.2)},
	{Currency: currency.EOSBULL, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.005, 0.01)},
	{Currency: currency.EPS, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.62, 1.24)},
	{Currency: currency.EPS, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.62, 1.24)},
	{Currency: currency.ERD, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0, 1)},
	{Currency: currency.ERD, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1000, 1500)},
	{Currency: currency.ERN, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(2.39, 4.78)},
	{Currency: currency.ETC, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.0061, 0.012)},
	{Currency: currency.ETC, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.0061, 0.012)},
	{Currency: currency.ETC, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.01, 0.02)},
	{Currency: currency.ETF, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1, 2)},
	{Currency: currency.ETH, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.002, 0.01)},
	{Currency: currency.ETH, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.000069, 0.00014)},
	{Currency: currency.ETH, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.000069, 0.00014)},
	{Currency: currency.ETH, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.005, 0.01)},
	{Currency: currency.EOSBEAR, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.022, 0.044)},
	{Currency: currency.EGLD, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.00065, 0.0013)},
	{Currency: currency.EGLD, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.001, 0.01)},
	{Currency: currency.EVX, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.44, 0.88)},
	{Currency: currency.EVX, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(51, 102)},
	{Currency: currency.ETHBEAR, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0, 0.0001)},
	{Currency: currency.ETHBNT, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(5, 5)},
	{Currency: currency.EDO, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(2.23, 4.46)},
	{Currency: currency.ENTRP, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(10, 20)},
	{Currency: currency.ENTRP, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(10, 20)},
	{Currency: currency.EFI, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0, 0)},
	{Currency: currency.FARM, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.0024, 0.0048)},
	{Currency: currency.FARM, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.28, 0.56)},
	{Currency: currency.FLM, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.5, 2)},
	{Currency: currency.FIDA, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.26, 0.52)},
	{Currency: currency.FOR, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(3.81, 7.62)},
	{Currency: currency.FOR, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(443, 886)},
	{Currency: currency.FLOW, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.023, 0.046)},
	{Currency: currency.FLOW, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.01, 2.7)},
	{Currency: currency.FTM, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.12, 0.24)},
	{Currency: currency.FTM, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.12, 0.24)},
	{Currency: currency.FTM, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(13, 26)},
	{Currency: currency.FTM, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.01, 1)},
	{Currency: currency.FTT, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.68, 1.36)},
	{Currency: currency.FUN, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1699, 3398)},
	{Currency: currency.FUEL, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(564, 1128)},
	{Currency: currency.FXS, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.016, 0.032)},
	{Currency: currency.FXS, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1.85, 3.7)},
	{Currency: currency.FRONT, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.3, 0.6)},
	{Currency: currency.FRONT, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(34, 68)},
	{Currency: currency.FIRO, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.037, 0.074)},
	{Currency: currency.FIRO, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.02, 0.04)},
	{Currency: currency.FORTH, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(6.25, 12)},
	{Currency: currency.FET, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.37, 0.74)},
	{Currency: currency.FET, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(43, 86)},
	{Currency: currency.FET, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.5, 1.5)},
	{Currency: currency.FIL, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.0058, 0.012)},
	{Currency: currency.FIL, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.0058, 0.012)},
	{Currency: currency.FIL, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.67, 1.34)},
	{Currency: currency.FIL, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.001, 0.01)},
	{Currency: currency.FIO, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(5, 10)},
	{Currency: currency.FIS, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.18, 0.36)},
	{Currency: currency.FIS, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(20, 40)},
	{Currency: currency.GLM, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(66, 132)},
	{Currency: currency.GNO, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.083, 0.17)},
	{Currency: currency.GNT, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0, 20)},
	{Currency: currency.GRT, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(39, 78)},
	{Currency: currency.GRS, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.2, 0.4)},
	{Currency: currency.GO, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1, 2)},
	{Currency: currency.GTC, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(6.71, 13)},
	{Currency: currency.GTO, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(4.68, 9.36)},
	{Currency: currency.GTO, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(540, 1080)},
	{Currency: currency.GYEN, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1239, 2478)},
	{Currency: currency.GVT, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.29, 0.58)},
	{Currency: currency.GVT, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(33, 66)},
	{Currency: currency.GXS, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.3, 0.6)},
	{Currency: currency.GAS, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.005, 1)},
	{Currency: currency.GAS, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.05, 0.1)},
	{Currency: currency.GHST, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(12, 24)},
	{Currency: currency.GALA, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.56, 1.12)},
	{Currency: currency.GALA, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(64, 128)},
	{Currency: currency.HNT, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.05, 1)},
	{Currency: currency.HOT, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(2571, 5142)},
	{Currency: currency.HARD, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.28, 0.56)},
	{Currency: currency.HARD, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.1, 0.2)},
	{Currency: currency.HC, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.005, 0.01)},
	{Currency: currency.HEGIC, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(312, 624)},
	{Currency: currency.HNST, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(10, 20)},
	{Currency: currency.HNST, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(10, 20)},
	{Currency: currency.HBAR, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1, 2)},
	{Currency: currency.HCC, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.0005, 0.0005)},
	{Currency: currency.HIVE, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.01, 0.02)},
	{Currency: currency.IRIS, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1, 2)},
	{Currency: currency.IDRT, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(4275, 8550)},
	{Currency: currency.IDRT, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(494742, 989484)},
	{Currency: currency.IQ, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(50, 100)},
	{Currency: currency.IOTX, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1.65, 3.3)},
	{Currency: currency.IOTX, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1.65, 3.3)},
	{Currency: currency.IOTX, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0, 10)},
	{Currency: currency.IOTX, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.1, 1)},
	{Currency: currency.IOST, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(849, 1698)},
	{Currency: currency.IOST, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1, 2)},
	{Currency: currency.IOTA, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.2, 0.4)},
	{Currency: currency.IOTA, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.5, 1.5)},
	{Currency: currency.ICP, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.0003, 0.001)},
	{Currency: currency.ICX, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.02, 0.04)},
	{Currency: currency.IDEX, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.78, 1.56)},
	{Currency: currency.IDEX, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(90, 180)},
	{Currency: currency.ILV, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.00018, 0.00036)},
	{Currency: currency.ILV, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.02, 0.04)},
	{Currency: currency.INJ, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.023, 0.046)},
	{Currency: currency.INJ, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.023, 0.046)},
	{Currency: currency.INJ, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1.13, 2.26)},
	{Currency: currency.INS, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(52, 104)},
	{Currency: currency.JST, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(3.87, 7.74)},
	{Currency: currency.JST, Chain: "TRC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(50, 100)},
	{Currency: currency.JUV, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.01, 0.02)},
	{Currency: currency.JUV, Chain: "CHZ", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0, 1)},
	{Currency: currency.JASMY, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(191, 382)},
	{Currency: currency.JEX, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(10.85, 21.7)},
	{Currency: currency.KNCL, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(4, 8)},
	{Currency: currency.KLAY, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.005, 0.01)},
	{Currency: currency.KEY, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(2185, 4370)},
	{Currency: currency.KP3R, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.029, 0.058)},
	{Currency: currency.KAVA, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.056, 0.11)},
	{Currency: currency.KAVA, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.001, 2)},
	{Currency: currency.KEYFI, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(17, 34)},
	{Currency: currency.KMD, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.31, 0.62)},
	{Currency: currency.KMD, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.002, 0.004)},
	{Currency: currency.KNC, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.17, 0.34)},
	{Currency: currency.KNC, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(4, 8)},
	{Currency: currency.KEEP, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(42, 84)},
	{Currency: currency.KSM, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.0009, 0.0018)},
	{Currency: currency.KSM, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.01, 1.01)},
	{Currency: currency.LOOMOLD, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0, 230)},
	{Currency: currency.LEND, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1, 2)},
	{Currency: currency.LUNA, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.02, 2.3)},
	{Currency: currency.LAZIO, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.048, 0.096)},
	{Currency: currency.LBA, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(10, 20)},
	{Currency: currency.LBA, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(10, 20)},
	{Currency: currency.LIT, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.068, 0.14)},
	{Currency: currency.LIT, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(7.92, 15)},
	{Currency: currency.LOOM, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(2.29, 4.58)},
	{Currency: currency.LOOM, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(264, 528)},
	{Currency: currency.LLT, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(100, 200)},
	{Currency: currency.LPT, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.72, 1.44)},
	{Currency: currency.LRC, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(9.87, 19)},
	{Currency: currency.LSK, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.1, 0.2)},
	{Currency: currency.LTC, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.0014, 0.0028)},
	{Currency: currency.LTC, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.0014, 0.0028)},
	{Currency: currency.LTC, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.001, 0.002)},
	{Currency: currency.LTO, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.51, 1.02)},
	{Currency: currency.LTO, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.51, 1.02)},
	{Currency: currency.LTO, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(58, 116)},
	{Currency: currency.LTO, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(5, 10)},
	{Currency: currency.LINA, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(4.99, 9.98)},
	{Currency: currency.LINA, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(866, 1732)},
	{Currency: currency.LINK, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.002, 0.01)},
	{Currency: currency.LINK, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.011, 0.022)},
	{Currency: currency.LINK, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.512, 1.024)},
	{Currency: currency.LUN, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0, 3.08)},
	{Currency: currency.MINA, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.5, 2.5)},
	{Currency: currency.MITH, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(2.78, 5.56)},
	{Currency: currency.MITH, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(321, 642)},
	{Currency: currency.MBL, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(3.52, 7.04)},
	{Currency: currency.MTLX, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1.5, 3)},
	{Currency: currency.MCO, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1.41, 2.82)},
	{Currency: currency.MDA, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.44, 0.88)},
	{Currency: currency.MDA, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(50, 100)},
	{Currency: currency.MA, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0, 0)},
	{Currency: currency.MDT, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(4.81, 9.62)},
	{Currency: currency.MDT, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(555, 1110)},
	{Currency: currency.MDX, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.42, 0.84)},
	{Currency: currency.MFT, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(2259, 4518)},
	{Currency: currency.MATIC, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.17, 0.34)},
	{Currency: currency.MATIC, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.17, 0.34)},
	{Currency: currency.MATIC, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(9.72, 19.44)},
	{Currency: currency.MATIC, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.1, 0.2)},
	{Currency: currency.MBOX, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.041, 0.082)},
	{Currency: currency.MIR, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.11, 0.22)},
	{Currency: currency.MIR, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(19, 38)},
	{Currency: currency.MANA, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(7.33, 14)},
	{Currency: currency.MEETONE, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(300, 600)},
	{Currency: currency.MKR, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.000094, 0.00019)},
	{Currency: currency.MKR, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.000094, 0.00019)},
	{Currency: currency.MKR, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.011, 0.022)},
	{Currency: currency.MLN, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.01, 0.02)},
	{Currency: currency.MASK, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.022, 0.044)},
	{Currency: currency.MASK, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(2.53, 5.06)},
	{Currency: currency.MOD, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(5, 10)},
	{Currency: currency.MDXT, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(100, 200)},
	{Currency: currency.MTH, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(959, 1918)},
	{Currency: currency.MTL, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(11, 22)},
	{Currency: currency.MOVR, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.01, 0.02)},
	{Currency: currency.NEAR, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.032, 0.064)},
	{Currency: currency.NEAR, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.032, 0.064)},
	{Currency: currency.NEAR, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.01, 0.2)},
	{Currency: currency.NPXS, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(4968, 9936)},
	{Currency: currency.NEBL, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.01, 0.02)},
	{Currency: currency.NCASH, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(11108, 22216)},
	{Currency: currency.NSBT, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.002, 0.004)},
	{Currency: currency.NAS, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.1, 0.2)},
	{Currency: currency.NAV, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.79, 1.58)},
	{Currency: currency.NAV, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.2, 0.4)},
	{Currency: currency.NBS, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1, 2)},
	{Currency: currency.NEO, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0, 1)},
	{Currency: currency.NEO, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0, 1)},
	{Currency: currency.NFT, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(44295, 88590)},
	{Currency: currency.NFT, Chain: "TRC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(16800, 33600)},
	{Currency: currency.NULS, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.56, 1.12)},
	{Currency: currency.NULS, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.01, 0.02)},
	{Currency: currency.NU, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(41, 82)},
	{Currency: currency.NKN, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(65, 130)},
	{Currency: currency.NMR, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.86, 1.72)},
	{Currency: currency.NANO, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.01, 0.02)},
	{Currency: currency.NVT, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.1, 1)},
	{Currency: currency.NXS, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.02, 0.04)},
	{Currency: currency.OCEAN, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.3, 0.6)},
	{Currency: currency.OCEAN, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(34, 68)},
	{Currency: currency.OAX, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1.2, 2.4)},
	{Currency: currency.OAX, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(138, 276)},
	{Currency: currency.OGN, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(29, 58)},
	{Currency: currency.OG, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.02, 0.04)},
	{Currency: currency.OG, Chain: "CHZ", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0, 1)},
	{Currency: currency.OM, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1.27, 2.54)},
	{Currency: currency.OM, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(146, 292)},
	{Currency: currency.OMG, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(3.94, 7.88)},
	{Currency: currency.ONE, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0, 8)},
	{Currency: currency.ONE, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.002, 60)},
	{Currency: currency.ONG, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.043, 0.086)},
	{Currency: currency.ONT, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.31, 0.62)},
	{Currency: currency.ONT, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.31, 0.62)},
	{Currency: currency.ONT, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1, 2)},
	{Currency: currency.ONX, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(4.21, 8.42)},
	{Currency: currency.ORN, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.039, 0.078)},
	{Currency: currency.ORN, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(4.53, 9.06)},
	{Currency: currency.OST, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(2825, 5650)},
	{Currency: currency.OMOLD, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(41, 82)},
	{Currency: currency.OXT, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(71, 142)},
	{Currency: currency.PAXG, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.019, 0.038)},
	{Currency: currency.PAX, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0, 10)},
	{Currency: currency.PAX, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0, 10)},
	{Currency: currency.PAX, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(15, 30)},
	{Currency: currency.POWR, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(47, 94)},
	{Currency: currency.PHA, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.46, 0.92)},
	{Currency: currency.PHA, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(16, 32)},
	{Currency: currency.PHB, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.45, 0.9)},
	{Currency: currency.PUNDIX, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(24, 48)},
	{Currency: currency.PLA, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(8, 16)},
	{Currency: currency.PLA, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.058, 0.12)},
	{Currency: currency.PHBV1, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(36, 72)},
	{Currency: currency.PHBV1, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(36, 72)},
	{Currency: currency.PHBV1, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(3, 6)},
	{Currency: currency.PNT, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.26, 0.52)},
	{Currency: currency.PNT, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(29, 58)},
	{Currency: currency.POA, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1449, 2898)},
	{Currency: currency.POA, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.01, 0.02)},
	{Currency: currency.POE, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(63944, 127888)},
	{Currency: currency.PPT, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(28, 56)},
	{Currency: currency.PIVX, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.2, 0.4)},
	{Currency: currency.PSG, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.01, 0.02)},
	{Currency: currency.PSG, Chain: "CHZ", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0, 1)},
	{Currency: currency.PERL, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(2.91, 5.82)},
	{Currency: currency.PERL, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(335, 670)},
	{Currency: currency.PERP, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.021, 0.042)},
	{Currency: currency.PERP, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(3.65, 7.3)},
	{Currency: currency.PERLOLD, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0, 10)},
	{Currency: currency.PROS, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.15, 0.3)},
	{Currency: currency.PROS, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(16, 32)},
	{Currency: currency.PROM, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.018, 0.036)},
	{Currency: currency.PROM, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(2.13, 4.26)},
	{Currency: currency.PORTO, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.5, 1)},
	{Currency: currency.PARA, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0, 0)},
	{Currency: currency.POND, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(3.1, 6.2)},
	{Currency: currency.POND, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(358, 716)},
	{Currency: currency.POLS, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.082, 0.16)},
	{Currency: currency.POLS, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(9.45, 18)},
	{Currency: currency.POLY, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(48, 96)},
	{Currency: currency.QUICK, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.11, 0.22)},
	{Currency: currency.QUICK, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.00053, 0.0011)},
	{Currency: currency.QKC, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(10, 20)},
	{Currency: currency.QKC, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1166, 2332)},
	{Currency: currency.QLC, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1, 2)},
	{Currency: currency.QISWAP, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1.64, 3.28)},
	{Currency: currency.QNT, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.15, 0.3)},
	{Currency: currency.QI, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(5, 10)},
	{Currency: currency.QI, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1.29, 2.58)},
	{Currency: currency.QSP, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(583, 1166)},
	{Currency: currency.QTUM, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.01, 0.02)},
	{Currency: currency.RENBTC, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.0000052, 0.00001)},
	{Currency: currency.RENBTC, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.0006, 0.0012)},
	{Currency: currency.RAY, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.21, 0.42)},
	{Currency: currency.RAMP, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.8, 1.6)},
	{Currency: currency.RAMP, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(91, 182)},
	{Currency: currency.RCN, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(539, 1078)},
	{Currency: currency.RDN, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(64, 128)},
	{Currency: currency.REN, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(39, 78)},
	{Currency: currency.REP, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1.57, 3.14)},
	{Currency: currency.REQ, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(156, 312)},
	{Currency: currency.RARE, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(17, 34)},
	{Currency: currency.RGT, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1.13, 2.26)},
	{Currency: currency.RIF, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(3, 6)},
	{Currency: currency.RLC, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(8.7, 17)},
	{Currency: currency.ROSE, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.1, 1)},
	{Currency: currency.RSR, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(943, 1886)},
	{Currency: currency.REEF, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(11, 22)},
	{Currency: currency.REEF, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1321, 2642)},
	{Currency: currency.RVN, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1, 2)},
	{Currency: currency.RUNE, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.027, 0.054)},
	{Currency: currency.REPV1, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.1, 0.2)},
	{Currency: currency.RAD, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(2.94, 5.88)},
	{Currency: currency.STPT, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(243, 486)},
	{Currency: currency.SPARTA, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(2, 4)},
	{Currency: currency.SUSD, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(15, 30)},
	{Currency: currency.SFP, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.17, 0.34)},
	{Currency: currency.SLPOLD, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0, 614)},
	{Currency: currency.SALT, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(5.2, 10.4)},
	{Currency: currency.STORM, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(100, 200)},
	{Currency: currency.STORJ, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(16, 32)},
	{Currency: currency.SGT, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(200, 400)},
	{Currency: currency.SAND, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(5, 10)},
	{Currency: currency.SCRT, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.042, 0.084)},
	{Currency: currency.SCRT, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.1, 0.2)},
	{Currency: currency.SBTC, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.0005, 0.001)},
	{Currency: currency.SKL, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(126, 252)},
	{Currency: currency.SKY, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.02, 0.04)},
	{Currency: currency.STEEM, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.01, 0.02)},
	{Currency: currency.SLP, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(4.72, 9.44)},
	{Currency: currency.SLP, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(545, 1090)},
	{Currency: currency.SLP, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1, 154)},
	{Currency: currency.SNM, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.64, 1.28)},
	{Currency: currency.SNM, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(74, 148)},
	{Currency: currency.SNT, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(383, 766)},
	{Currency: currency.SNX, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.037, 0.074)},
	{Currency: currency.SNX, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.037, 0.074)},
	{Currency: currency.SNX, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(7.58, 15)},
	{Currency: currency.SNMOLD, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(329, 658)},
	{Currency: currency.SOL, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.0014, 0.0028)},
	{Currency: currency.SOL, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.01, 0.3)},
	{Currency: currency.SRM, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(6.19, 12)},
	{Currency: currency.SRM, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.38, 0.76)},
	{Currency: currency.SSV, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(2.87, 5.74)},
	{Currency: currency.SHIB, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(7232, 14464)},
	{Currency: currency.SHIB, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(830558, 1661116)},
	{Currency: currency.STX, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1.5, 5)},
	{Currency: currency.SUB, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0, 40)},
	{Currency: currency.SUN, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(9.7, 19)},
	{Currency: currency.SUN, Chain: "TRC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(3.19, 6.38)},
	{Currency: currency.SUNOLD, Chain: "TRC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.0027, 0.0058)},
	{Currency: currency.SPARTAOLD, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(2, 4)},
	{Currency: currency.SUSHI, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.037, 0.074)},
	{Currency: currency.SUSHI, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.037, 0.074)},
	{Currency: currency.SUSHI, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(4.28, 8.56)},
	{Currency: currency.SC, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.1, 0.2)},
	{Currency: currency.SXP, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.12, 0.24)},
	{Currency: currency.SXP, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.12, 0.24)},
	{Currency: currency.SXP, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(14, 28)},
	{Currency: currency.SYS, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1, 2)},
	{Currency: currency.STRAX, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.1, 0.2)},
	{Currency: currency.SNGLS, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1843, 3737)},
	{Currency: currency.SWRV, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(19, 38)},
	{Currency: currency.SUPER, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.17, 0.34)},
	{Currency: currency.SUPER, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(29, 58)},
	{Currency: currency.STMX, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1081, 2162)},
	{Currency: currency.TRIG, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(50, 51)},
	{Currency: currency.TUSD, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1, 10)},
	{Currency: currency.TUSD, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.3, 0.6)},
	{Currency: currency.TUSD, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(15, 30)},
	{Currency: currency.TKO, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.18, 0.36)},
	{Currency: currency.TKO, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.18, 0.36)},
	{Currency: currency.TFUEL, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(2.52, 5.04)},
	{Currency: currency.TROY, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(21, 42)},
	{Currency: currency.TROY, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(2509, 5018)},
	{Currency: currency.TLM, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.64, 1.28)},
	{Currency: currency.TLM, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(111, 222)},
	{Currency: currency.TNT, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(2070, 4140)},
	{Currency: currency.TOMO, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.092, 0.18)},
	{Currency: currency.TOMO, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(10, 20)},
	{Currency: currency.TOMO, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.01, 0.02)},
	{Currency: currency.TRB, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.61, 1.22)},
	{Currency: currency.TRU, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.67, 1.34)},
	{Currency: currency.TRU, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(77, 154)},
	{Currency: currency.TRX, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(3.04, 6.08)},
	{Currency: currency.TRX, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(3.02, 6.04)},
	{Currency: currency.TRX, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(351, 702)},
	{Currency: currency.TRX, Chain: "TRC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1, 2)},
	{Currency: currency.TORN, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.0058, 0.012)},
	{Currency: currency.TORN, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.67, 1.34)},
	{Currency: currency.TVK, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(79, 158)},
	{Currency: currency.TWT, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.26, 0.52)},
	{Currency: currency.TWT, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.26, 0.52)},
	{Currency: currency.THETA, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.12, 0.24)},
	{Currency: currency.TBCC, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(5, 10)},
	{Currency: currency.TCT, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(8.04, 16)},
	{Currency: currency.TCT, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(8.03, 16)},
	{Currency: currency.TCT, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(929, 1858)},
	{Currency: currency.TRIBE, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(31, 62)},
	{Currency: currency.UMA, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(2.49, 4.98)},
	{Currency: currency.UND, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(5, 10)},
	{Currency: currency.UND, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(5, 10)},
	{Currency: currency.UNI, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.014, 0.028)},
	{Currency: currency.UNI, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.014, 0.028)},
	{Currency: currency.UNI, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1.65, 3.3)},
	{Currency: currency.UNFI, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.028, 0.056)},
	{Currency: currency.UNFI, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(3.25, 6.5)},
	{Currency: currency.UTK, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(88, 176)},
	{Currency: currency.USDT, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1, 10)},
	{Currency: currency.USDT, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.8, 10)},
	{Currency: currency.USDT, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(24, 50)},
	{Currency: currency.USDT, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1, 10)},
	{Currency: currency.USDT, Chain: "TRC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1, 10)},
	{Currency: currency.USDS, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0, 0)},
	{Currency: currency.USDS, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(2, 4)},
	{Currency: currency.USDP, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.8, 10)},
	{Currency: currency.USDP, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(25, 50)},
	{Currency: currency.USDC, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1, 10)},
	{Currency: currency.USDC, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.8, 10)},
	{Currency: currency.USDC, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(25, 50)},
	{Currency: currency.USDC, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1, 10)},
	{Currency: currency.USDC, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1, 10)},
	{Currency: currency.UFT, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.33, 0.66)},
	{Currency: currency.UFT, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(38, 76)},
	{Currency: currency.VITE, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(2.62, 5.24)},
	{Currency: currency.VITE, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1, 2)},
	{Currency: currency.VTHO, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(200, 400)},
	{Currency: currency.VRT, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(47, 94)},
	{Currency: currency.VIDT, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.33, 0.66)},
	{Currency: currency.VIDT, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(37, 74)},
	{Currency: currency.VAB, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(142, 284)},
	{Currency: currency.VAI, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.8, 10)},
	{Currency: currency.VET, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(20, 40)},
	{Currency: currency.VGX, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(7.78, 15)},
	{Currency: currency.VIB, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(639, 1278)},
	{Currency: currency.VIA, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.01, 0.02)},
	{Currency: currency.VRAB, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(100, 200)},
	{Currency: currency.VRAB, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(300, 600)},
	{Currency: currency.WING, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.0023, 0.0046)},
	{Currency: currency.WNXM, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.42, 0.84)},
	{Currency: currency.WAVES, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.002, 0.004)},
	{Currency: currency.WPR, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1525, 3050)},
	{Currency: currency.WABI, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(152, 304)},
	{Currency: currency.WRX, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.25, 0.5)},
	{Currency: currency.WRX, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.25, 0.5)},
	{Currency: currency.WRX, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(29, 58)},
	{Currency: currency.WTC, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(36, 72)},
	{Currency: currency.WTC, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.01, 1)},
	{Currency: currency.WINGS, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0, 40)},
	{Currency: currency.WSOL, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.01, 0.3)},
	{Currency: currency.WBNB, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.002, 0.01)},
	{Currency: currency.WETH, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.01, 0.02)},
	{Currency: currency.WETH, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.001, 0.01)},
	{Currency: currency.WBTC, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.00059, 0.0012)},
	{Currency: currency.WAN, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.1, 0.2)},
	{Currency: currency.WAXP, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1.5, 3)},
	{Currency: currency.WIN, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(392, 784)},
	{Currency: currency.WIN, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(389, 778)},
	{Currency: currency.WIN, Chain: "TRC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(128, 256)},
	{Currency: currency.XPR, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1, 2)},
	{Currency: currency.XRP, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.29, 0.58)},
	{Currency: currency.XRP, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.29, 0.58)},
	{Currency: currency.XRP, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.25, 30)},
	{Currency: currency.XTZ, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.061, 0.12)},
	{Currency: currency.XTZ, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.02, 0.1)},
	{Currency: currency.XTZ, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.1, 1)},
	{Currency: currency.XDATA, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(143, 286)},
	{Currency: currency.XVG, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.1, 0.2)},
	{Currency: currency.XVS, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.012, 0.024)},
	{Currency: currency.XVS, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.012, 0.024)},
	{Currency: currency.XRPBULL, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.076, 0.15)},
	{Currency: currency.XYM, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.1, 0.2)},
	{Currency: currency.XRPBEAR, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.0012, 0.0024)},
	{Currency: currency.XEC, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1892, 3784)},
	{Currency: currency.XEC, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(10000, 20000)},
	{Currency: currency.XEM, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(4, 8)},
	{Currency: currency.XLM, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.9, 1.8)},
	{Currency: currency.XLM, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.02, 10)},
	{Currency: currency.XMR, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.0001, 0.0002)},
	{Currency: currency.YOYO, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1, 2)},
	{Currency: currency.YFII, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.000079, 0.00016)},
	{Currency: currency.YFII, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.000079, 0.00016)},
	{Currency: currency.YFII, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.0092, 0.018)},
	{Currency: currency.YFI, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.0000095, 0.000019)},
	{Currency: currency.YFI, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.0000095, 0.000019)},
	{Currency: currency.YFI, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.0011, 0.0022)},
	{Currency: currency.YGG, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.034, 0.068)},
	{Currency: currency.YGG, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(3.94, 7.88)},
	{Currency: currency.ZRX, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(30, 60)},
	{Currency: currency.ZCX, Chain: "ERC20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(1, 2)},
	{Currency: currency.ZEC, Chain: "BEP2", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.0012, 0.0024)},
	{Currency: currency.ZEC, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.0012, 0.0024)},
	{Currency: currency.ZEC, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.001, 0.01)},
	{Currency: currency.ZEN, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.002, 0.004)},
	{Currency: currency.ZIL, Chain: "BEP20", Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(3.38, 6.76)},
	{Currency: currency.ZIL, Deposit: fee.Convert(0), Withdrawal: fee.ConvertWithMinimumAmount(0.2, 0.4)},
}
