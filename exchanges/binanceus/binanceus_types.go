package binanceus

import (
	"strconv"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
)

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
	// Servertime uint64 `json:"serverTime"`
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

// RecentTradeRequestParams represents Klines request data.
type RecentTradeRequestParams struct {
	Symbol currency.Pair `json:"symbol"` // Required field. example LTCBTC, BTCUSDT
	Limit  int           `json:"limit"`  // Default 500; max 1000.
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

type HistoricalTradeParams struct {
	Symbol string `json:"symbol"`  // Required field. example LTCBTC, BTCUSDT
	Limit  int    `json:"limit"`   // Default 500; max 1000.
	FromID uint64 `json:"from_id"` // Optional Field. Specifies the trade ID to fetch most recent trade histories from
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
	StartTime uint64
	EndTime   uint64
	// Default 500; max 1000.
	Limit int
}

// toTradeData this method converts the AggregatedTrade data into an instance of trade.Data...
func (a *AggregatedTrade) toTradeData(p currency.Pair, exchange string, aType asset.Item) *trade.Data {
	return &trade.Data{
		CurrencyPair: p,
		TID:          strconv.FormatInt(a.ATradeID, 10),
		Amount:       a.Quantity,
		Exchange:     exchange,
		Price:        a.Price,
		Timestamp:    a.TimeStamp,
		AssetType:    aType,
		Side:         order.AnySide,
	}
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
	LastUpdateID int64 `json:"lastUpdateId"`
	// Code         int         `json:"code"`
	// Msg          string      `json:"msg"`
	Bids [][2]string `json:"bids"`
	Asks [][2]string `json:"asks"`
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

// KlinesRequestParams represents Klines request data.
type KlinesRequestParams struct {
	Symbol    currency.Pair // Required field; example LTCBTC, BTCUSDT
	Interval  string        // Time interval period
	Limit     int           // Default 500; max 500.
	StartTime time.Time
	EndTime   time.Time
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

type SymbolPrice struct {
	Symbol string  `json:"symbol"`
	Price  float64 `json:"price,string"`
}

type SymbolPrices []SymbolPrice

// AveragePrice holds current average symbol price
type AveragePrice struct {
	Mins  int64   `json:"mins"`
	Price float64 `json:"price,string"`
}

// BestPrice holds best price data
type BestPrice struct {
	Symbol   string  `json:"symbol"`
	BidPrice float64 `json:"bidPrice,string"`
	BidQty   float64 `json:"bidQty,string"`
	AskPrice float64 `json:"askPrice,string"`
	AskQty   float64 `json:"askQty,string"`
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

// Response holds basic binance api response data
type Response struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
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
	AccounType       string    `json:"spot"`
	Balances         []Balance `json:"balances"`
	Permissions      []string  `json:"permissions"`
}

// Balance holds query order data
type Balance struct {
	Asset  string          `json:"asset"`
	Free   decimal.Decimal `json:"free"`
	Locked decimal.Decimal `json:"locked"`
}

// AccountStatusResponse holds informations related to the
// User Account status information request
type AccountStatusResponse struct {
	Msg     string   `json:"msg"`
	Success bool     `json:"success"`
	Objs    []string `json:"objs,omitempty"`
}

type TradeStatus struct {
	IsLocked           bool                                  `json:"isLocked"`
	PlannedRecoverTime uint                                  `json:"plannedRecoverTime"`
	TriggerCondition   map[string]uint                       `json:"triggerCondition"`
	Indicators         map[string]TradingStatusIndicatorItem `json:"indicators"`
	UpdateTime         time.Time                             `json:"updateTime"`
}

type TradingStatusIndicatorItem struct {
	I string  `json:"i"`
	C float32 `json:"c"`
	V float32 `json:"v"`
	T float32 `json:"t"`
}

type TradeFee struct {
	Symbol string  `json:"symbol"`
	Maker  float64 `json:"maker"`
	Taker  float64 `json:"taker"`
}

type TradeFeeList struct {
	TradeFee []TradeFee `json:"tradeFee"`
	Success  bool       `json:"success,omitempty"`
}

// AssetHistory
type AssetHistory struct {
	Amount  float64 `json:"amount,string"` // Amount
	Asset   string  `json:"asset"`         // Asset Type eg. BHFT
	DivTime uint64  `json:"divTime"`       // DivTime
	EnInfo  string  `json:"enInfo"`        //
	TranID  uint64  `json:"tranId"`        // Transaction ID
}

// AssetDictributionHistories this endpoint to query asset distribution records,
// including for staking, referrals and airdrops etc.
type AssetDistributionHistories struct {
	Rows  []*AssetHistory `json:"rows"`
	Total uint            `json:"total"`
}

// SubAccount  ...
type SubAccount struct {
	Email      string    `json:"email"`
	Status     bool      `json:"status"`
	Activated  bool      `json:"activated"`
	Mobile     string    `json:"mobile"`
	GAuth      bool      `json:"gAuth"`
	CreateTime time.Time `json:"createTime"`
}

// TransferHistory a single asset transfer history between Sub accounts
type TransferHistory struct {
	Fron      string    `json:"from"`
	To        string    `json:"to"`
	Asset     string    `json:"asset"`
	Qty       uint      `json:"qty,string"`
	TimeStamp time.Time `json:"time"`
}

// SubAccountTransferRequestParams used to transfer an asset from one account to another
// this
type SubaccountTransferRequestParams struct {
	FromEmail  string  // Mendatory
	ToEmail    string  // Mendatory
	Asset      string  // Mendatory
	Amount     float64 // Mendatory
	RecvWindow uint64
}

// SubAccountTransferResponse
type SubaccountTransferResponse struct {
	Success bool   `json:"success"`
	TxnID   uint64 `json:"txnId,string"`
}

// SubaccountAsset holds asset informations.
type AssetInfo struct {
	Asset  string `json:"asset"`
	Free   uint64 `json:"free"`
	Locked uint64 `json:"locked"`
}

// SubAccountAssets holds all the balance and email of a subaccount
type SubAccountAssets struct {
	Balances        []AssetInfo `json:"balances"`
	Success         bool        `json:"success"`
	SubaccountEmail string      `json:"email,omitempty"`
}

// OrderRateLimit
type OrderRateLimit struct {
	RateLimitType string `json:"rateLimitType"`
	Interval      string `json:"interval"`
	IntervalNum   uint   `json:"intervalNum"`
	Limit         uint   `json:"limit"`
	Count         uint   `json:"count"`
}

// RequestParamsOrderType trade order type
type RequestParamsOrderType string

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
	Symbol          string    `json:"symbol"`
	OrderID         int64     `json:"orderId"`
	OrderListID     int8      `json:"orderListId"`
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
	// --
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	// --
	Fills []struct {
		Price           float64 `json:"price,string"`
		Qty             float64 `json:"qty,string"`
		Commission      float64 `json:"commission,string"`
		CommissionAsset string  `json:"commissionAsset"`
	} `json:"fills"`
}

// CommonOrder instance holds the order informations common to both
// for Order and OrderReportItem
type CommonOrder struct {
	Symbol        string `json:"symbol"`
	OrderID       uint64 `json:"orderId"`
	OrderListID   int8   `json:"orderListId"`
	ClientOrderID string `json:"clientOrderId"`

	Price               float64 `json:"price,string"`
	OrigQty             float64 `json:"origQty,string"`
	ExecutedQty         float64 `json:"executedQty,string"`
	CummulativeQuoteQty float64 `json:"cummulativeQuoteQty,string"`
	Status              string  `json:"status"`
	TimeInForce         string  `json:"timeInForce"`
	Type                string  `json:"type"`
	Side                string  `json:"side"`
	StopPrice           float64 `json:"stopPrice,string"`
}

// Order struct
type Order struct {
	CommonOrder
	IcebergQty        float64   `json:"icebergQty,string"`
	Time              time.Time `json:"time"`
	UpdateTime        time.Time `json:"updateTime"`
	IsWorking         bool      `json:"isWorking"`
	OrigQuoteOrderQty float64   `json:"origQuoteOrderQty"`
}

// OrderReportItem this is used by the OCO order creating response
type OCOOrderReportItem struct {
	CommonOrder
	TransactionTime time.Time `json:"transactionTime"`
}

// GetOrderRequestParams this struct will be used to get a
// order and its related information
type OrderRequestParams struct {
	Symbol            string `json:"symbol"` // REQUIRED
	OrderID           uint64 `json:"orderId"`
	OrigClientOrderId string `json:"origClientOrderId"`
	RecvWindow        uint
}

// CancelOrderRequestParams this struct will be used as a parameter for
// cancel order method.
type CancelOrderRequestParams struct {
	Symbol currency.Pair
	// SymbolString      string
	OrderID           uint64
	OrigClientOrderID string
	NewClientOrderID  string
	RecvWindow        uint
}

// GetTradesParams
type GetTradesParams struct {
	Symbol     string     `json:"symbol"`
	OrderID    uint64     `json:"orderId"`
	StartTime  *time.Time `json:"startTime"`
	EndTime    *time.Time `json:"endTime"`
	FromID     uint64     `json:"fromId"`
	Limit      uint       `json:"limit"`
	RecvWindow uint       `json:"recvWindow"`
}

// Trade this struct represents a trade response.
type Trade struct {
	Symbol          string    `json:"symbol"`
	ID              uint64    `json:"id"`
	OrderID         uint64    `json:"orderId"`
	OrderListId     int       `json:"orderListId"`
	Price           float64   `json:"price"`
	Qty             float64   `json:"qty"`
	QuoteQty        float64   `json:"quoteQty"`
	Commission      float64   `json:"commission"`
	CommissionAsset float64   `json:"commissionAsset"`
	Time            time.Time `json:"time"`
	IsBuyer         bool      `json:"isBuyer"`
	IsMaker         bool      `json:"isMaker"`
	IsBestMatch     bool      `json:"isBestMatch"`
}

// OCOOrderParams
// One -cancel-the-other order creation input Parameter
type OCOOrderInputParams struct {
	Symbol               string  `json:"symbol"`    // Required
	StopPrice            float64 `json:"stopPrice"` // Required
	Side                 string  `json:"side"`      // Required
	Quantity             float64 `json:"quantity"`  // Required
	Price                float64 `json:"price"`     // Required
	ListClientOrderID    string  `json:"listClientOrderId"`
	LimitClientOrderID   string  `json:"limitClientOrderId"`
	LimitIcebergQty      float64 `json:"limitIcebergQty"`
	StopClientOrderID    string  `json:"stopClientOrderId"`
	StopLimitPrice       float64 `json:"stopLimitPrice"`
	StopIcebergQty       float64 `json:"stopIcebergQty"`
	StopLimitTimeInForce string  `json:"stopLimitTimeInForce"`
	NewOrderRespType     string  `json:"newOrderRespType"`
	RecvWindow           uint64  `json:"recvWindow"`
}

// GetOCOPrderRequestParams
type GetOCOPrderRequestParams struct {
	OrderListID       uint64
	OrigClientOrderID string
}

type OrderShortResponse struct {
	Symbol        string `json:"symbol"`
	OrderID       uint64 `json:"orderId"`
	ClientOrderID string `json:"clientOrderId"`
}

// OCONewOrderResponse this model is to be used to fetch the respons of create new OCO order response
//
type OCOOrderResponse struct {
	OrderListId       int64                 `json:"orderListId"`
	ContingencyType   string                `json:"contingencyType"`
	ListStatusType    string                `json:"listStatusType"`
	ListOrderStatus   string                `json:"listOrderStatus"`
	ListClientOrderId string                `json:"listClientOrderId"`
	TransactionTime   time.Time             `json:"transactionTime"`
	Symbol            string                `json:"symbol"`
	Orders            []*OrderShortResponse `json:"orders"`
}

// CreateNewOrderResponse
type OCOFullOrderResponse struct {
	*OCOOrderResponse
	OrderReports []*OCOOrderReportItem `json:"orderReports"`
}

// OCOOrdersRequestParams
type OCOOrdersRequestParams struct {
	FromID     uint64
	StartTime  time.Time
	EndTime    time.Time
	Limit      uint
	RecvWindow uint
}

// OCOOrdersDeleteRequestParams
// holds the params to delete a new order
type OCOOrdersDeleteRequestParams struct {
	Symbol            string
	OrderListID       uint64
	ListClientOrderID string
	NewClientOrderID  string
	RecvWindow        uint
}

// OTC endpoinsts

// CoinPairInfo
type CoinPairInfo struct {
	FromCoin          string  `json:"fromCoin"`
	ToCoin            string  `json:"toCoin"`
	FromCoinMinAmount float64 `json:"fromCoinMinAmount,string"`
	FromCoinMaxAmount float64 `json:"fromCoinMaxAmount,string"`
	ToCoinMinAmount   float64 `json:"toCoinMinAmount,string"`
	ToCoinMaxAmount   float64 `json:"toCoinMaxAmount,string"`
}

// RequestQuoteParams
type RequestQuoteParams struct {
	FromCoin      string  `json:"fronCoin"`
	ToCoin        string  `json:"toCoin"`
	RequestCoin   string  `json:"requestCoin"`
	RequestAmount float64 `json:"requestAmount"`
}

// RequestQuote
type RequestQuote struct {
	QuoteId        string  `json:"quoteId"`
	Symbol         string  `json:"symbol"`
	Ratio          float64 `json:"ratio"`
	InverseRatio   float64 `json:"inverseRatio"`
	ValidTimestamp float64 `json:"validTimestamp"`
	ToAmount       float64 `json:"toAmount"`
	FromAmount     uint64  `json:"fromAmount"`
}

// OTCTradeOrderResponse
type OTCTradeOrderResponse struct {
	OrderID     uint64    `json:"orderId,string"`
	CreateTime  time.Time `json:"createTime"`
	OrderStatus string    `json:"orderStatus"`
}

// OTCTradeOrder
type OTCTradeOrder struct {
	QuoteID      string    `json:"quoteId"`
	OrderID      uint64    `json:"orderId,string"`
	OrderStatus  string    `json:"orderStatus"`
	FromCoin     string    `json:"fromCoin"`
	FromAmount   float64   `json:"fromAmount"`
	ToCoin       string    `json:"toCoin"`
	ToAmount     float64   `json:"toAmount"`
	Ratio        float64   `json:"ratio"`
	InverseRatio float64   `json:"inverseRatio"`
	CreateTime   time.Time `json:"createTime"`
}

// OTCTradeOrderParams ...
type OTCTradeOrderRequestParams struct {
	OrderId   string
	FromCoin  string
	ToCoin    string
	StartTime *time.Time
	EndTime   *time.Time
	Limit     int8
}

// Wallet Endpoints
//

// AssetWalletDetail ...
type AssetWalletDetail struct {
	Coin              string `json:"coin"`
	DepositAllEnable  bool   `json:"depositAllEnable"`
	WithdrawAllEnable bool   `json:"withdrawAllEnable"`
	Name              string `json:"name"`
	Free              string `json:"free"`
	Locked            string `json:"locked"`
	Freeze            string `json:"freeze"`
	Withdrawing       string `json:"withdrawing"`
	Ipoing            string `json:"ipoing"`
	Ipoable           string `json:"ipoable"`
	Storage           string `json:"storage"`
	IsLegalMoney      bool   `json:"isLegalMoney"`
	Trading           bool   `json:"trading"`
	NetworkList       []struct {
		Network                 string  `json:"network"`
		Coin                    string  `json:"coin"`
		WithdrawIntegerMultiple string  `json:"withdrawIntegerMultiple"`
		IsDefault               bool    `json:"isDefault"`
		DepositEnable           bool    `json:"depositEnable"`
		WithdrawEnable          bool    `json:"withdrawEnable"`
		DepositDesc             string  `json:"depositDesc"`
		WithdrawDesc            string  `json:"withdrawDesc"`
		Name                    string  `json:"name"`
		ResetAddressStatus      bool    `json:"resetAddressStatus"`
		WithdrawFee             float64 `json:"withdrawFee,string"`
		WithdrawMin             float64 `json:"withdrawMin,string"`
		WithdrawMax             float64 `json:"withdrawMax,string"`
		AddressRegex            string  `json:"addressRegex,omitempty"`
		MemoRegex               string  `json:"memoRegex,omitempty"`
		MinConfirm              int     `json:"minConfirm,omitempty"`
		UnLockConfirm           int     `json:"unLockConfirm,omitempty"`
	} `json:"networkList"`
}

// AssetWalletList .. list of asset wallet details
type AssetWalletList []AssetWalletDetail

// WithdrawalRequestParam represents the params for the
// input parameters of Withdraw Crypto
type WithdrawalRequestParam struct {
	Coin            string  `json:"coin"`
	Network         string  `json:"network"`
	WithdrawOrderId string  `json:"withdrawOrderId"` // Client ID for withdraw
	Address         string  `json:"address"`
	AddressTag      string  `json:"addressTag"`
	Amount          float64 `json:"amount"`
	RecvWindow      uint64  `json:"recvWindow"`
}

// WithdrawalResponse ...
type WithdrawalResponse struct {
	ID string `json:"id"`
}

// WithdrawStatusResponse defines a withdrawal status response
type WithdrawStatusResponse struct {
	ID             string  `json:"id"`
	Amount         float64 `json:"amount,string"`
	TransactionFee float64 `json:"transactionFee,string"`
	Coin           string  `json:"coin"`
	Status         int     `json:"status"`
	Address        string  `json:"address"`
	ApplyTime      string  `json:"applyTime"`
	Network        string  `json:"network"`
	TransferType   int     `json:"transferType"`
}

// FiatAssetRecord ...
type FiatAssetRecord struct {
	OrderID        string `json:"orderId"`
	PaymentAccount string `json:"paymentAccount"`
	PaymentChannel string `json:"paymentChannel"`
	PaymentMethod  string `json:"paymentMethod"`
	OrderStatus    string `json:"orderStatus"`
	Amount         string `json:"amount"`
	TransactionFee string `json:"transactionFee"`
	PlatformFee    string `json:"platformFee"`
}

// FiatWithdrawalHistory ...
type FiatAssetsHistory struct {
	AssetLogRecordList []FiatAssetRecord `json:"assetLogRecordList"`
}

// WithdrawFiatRequestParams ...
type WithdrawFiatRequestParams struct {
	PaymentChannel string
	PaymentMethod  string
	PaymentAccount string
	FiatCurrency   string
	Amount         float64
	RecvWindow     uint64
}

// FiatWithdrawalRequestParams ... to fetch your fiat (USD) withdrawal history.
type FiatWithdrawalRequestParams struct {
	FiatCurrency   string
	OrderId        string
	Offset         int
	PaymentChannel string
	PaymentMethod  string
	StartTime      time.Time
	EndTime        time.Time
}

// DepositAddress stores the deposit address info
type DepositAddress struct {
	Address string `json:"address"`
	Coin    string `json:"coin"`
	Tag     string `json:"tag"`
	URL     string `json:"url"`
}

// DepositHistory stores deposit history info
type DepositHistory struct {
	Amount       string `json:"amount"`
	Coin         string `json:"coin"`
	Network      string `json:"network"`
	Status       int    `json:"status"`
	Address      string `json:"address"`
	AddressTag   string `json:"addressTag"`
	TxID         string `json:"txId"`
	InsertTime   int64  `json:"insertTime"`
	TransferType int    `json:"transferType"`
	ConfirmTimes string `json:"confirmTimes"`
}

// UserAccountStream
type UserAccountStream struct {
	ListenKey string `json:"listenKey"`
}
