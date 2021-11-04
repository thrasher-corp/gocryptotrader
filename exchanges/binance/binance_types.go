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
