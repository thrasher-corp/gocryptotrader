package binanceus

import (
	"strconv"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/types"
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

// crypto withdrawals status codes description
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
	Code       int64      `json:"code"`
	Msg        string     `json:"msg"`
	Timezone   string     `json:"timezone"`
	ServerTime types.Time `json:"serverTime"`
	RateLimits []struct {
		RateLimitType string `json:"rateLimitType"`
		Interval      string `json:"interval"`
		Limit         int64  `json:"limit"`
	} `json:"rateLimits"`
	ExchangeFilters any `json:"exchangeFilters"`
	Symbols         []struct {
		Symbol                     string   `json:"symbol"`
		Status                     string   `json:"status"`
		BaseAsset                  string   `json:"baseAsset"`
		BaseAssetPrecision         int64    `json:"baseAssetPrecision"`
		QuoteAsset                 string   `json:"quoteAsset"`
		QuotePrecision             int64    `json:"quotePrecision"`
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
	Limit  int64         `json:"limit"`  // Default 500; max 1000.
}

// RecentTrade holds recent trade data
type RecentTrade struct {
	ID           int64      `json:"id"`
	Price        float64    `json:"price,string"`
	Quantity     float64    `json:"qty,string"`
	Time         types.Time `json:"time"`
	IsBuyerMaker bool       `json:"isBuyerMaker"`
	IsBestMatch  bool       `json:"isBestMatch"`
}

// HistoricalTradeParams represents historical trades request params.
type HistoricalTradeParams struct {
	Symbol string `json:"symbol"` // Required field. example LTCBTC, BTCUSDT
	Limit  int64  `json:"limit"`  // Default 500; max 1000.
	FromID uint64 `json:"fromId"` // Optional Field. Specifies the trade ID to fetch most recent trade histories from
}

// HistoricalTrade holds recent trade data
type HistoricalTrade struct {
	ID            int64      `json:"id"`
	Price         float64    `json:"price,string"`
	Quantity      float64    `json:"qty,string"`
	QuoteQuantity float64    `json:"quoteQty,string"`
	Time          types.Time `json:"time"`
	IsBuyerMaker  bool       `json:"isBuyerMaker"`
	IsBestMatch   bool       `json:"isBestMatch"`
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
	ATradeID       int64      `json:"a"`
	Price          float64    `json:"p,string"`
	Quantity       float64    `json:"q,string"`
	FirstTradeID   int64      `json:"f"`
	LastTradeID    int64      `json:"l"`
	TimeStamp      types.Time `json:"T"`
	Maker          bool       `json:"m"`
	BestMatchPrice bool       `json:"M"`
}

// toTradeData this method converts the AggregatedTrade data into an instance of trade.Data
func (a *AggregatedTrade) toTradeData(p currency.Pair, exchange string, aType asset.Item) *trade.Data {
	return &trade.Data{
		CurrencyPair: p,
		TID:          strconv.FormatInt(a.ATradeID, 10),
		Amount:       a.Quantity,
		Exchange:     exchange,
		Price:        a.Price,
		Timestamp:    a.TimeStamp.Time(),
		AssetType:    aType,
		Side:         order.AnySide,
	}
}

// OrderBookData is resp data from orderbook endpoint
type OrderBookData struct {
	LastUpdateID int64                            `json:"lastUpdateId"`
	Bids         orderbook.LevelsArrayPriceAmount `json:"bids"`
	Asks         orderbook.LevelsArrayPriceAmount `json:"asks"`
}

// OrderBook actual structured data that can be used for orderbook
type OrderBook struct {
	Symbol       string
	LastUpdateID int64
	Code         int
	Msg          string
	Bids         []orderbook.Level
	Asks         []orderbook.Level
}

// KlinesRequestParams represents Klines request data.
type KlinesRequestParams struct {
	Symbol    currency.Pair // Required field; example LTCBTC, BTCUSDT
	Interval  string        // Time interval period
	Limit     uint64        // Default 500; max 500.
	StartTime time.Time
	EndTime   time.Time
}

// CandleStick holds kline data
type CandleStick struct {
	OpenTime                 types.Time
	Open                     types.Number
	High                     types.Number
	Low                      types.Number
	Close                    types.Number
	Volume                   types.Number
	CloseTime                types.Time
	QuoteAssetVolume         types.Number
	TradeCount               types.Number
	TakerBuyAssetVolume      types.Number
	TakerBuyQuoteAssetVolume types.Number
}

// UnmarshalJSON unmarshals JSON data into a CandleStick struct
func (c *CandleStick) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[11]any{&c.OpenTime, &c.Open, &c.High, &c.Low, &c.Close, &c.Volume, &c.CloseTime, &c.QuoteAssetVolume, &c.TradeCount, &c.TakerBuyAssetVolume, &c.TakerBuyQuoteAssetVolume})
}

// SymbolPrice represents a symbol and it's price.
type SymbolPrice struct {
	Symbol string  `json:"symbol"`
	Price  float64 `json:"price,string"`
}

// SymbolPrices lis tof Symbol Price
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
	Symbol             string     `json:"symbol"`
	PriceChange        float64    `json:"priceChange,string"`
	PriceChangePercent float64    `json:"priceChangePercent,string"`
	WeightedAvgPrice   float64    `json:"weightedAvgPrice,string"`
	PrevClosePrice     float64    `json:"prevClosePrice,string"`
	LastPrice          float64    `json:"lastPrice,string"`
	LastQty            float64    `json:"lastQty,string"`
	BidPrice           float64    `json:"bidPrice,string"`
	AskPrice           float64    `json:"askPrice,string"`
	OpenPrice          float64    `json:"openPrice,string"`
	HighPrice          float64    `json:"highPrice,string"`
	LowPrice           float64    `json:"lowPrice,string"`
	Volume             float64    `json:"volume,string"`
	QuoteVolume        float64    `json:"quoteVolume,string"`
	OpenTime           types.Time `json:"openTime"`
	CloseTime          types.Time `json:"closeTime"`
	FirstID            int64      `json:"firstId"`
	LastID             int64      `json:"lastId"`
	Count              int64      `json:"count"`
}

// Response holds basic binance api response data
type Response struct {
	Code int64  `json:"code"`
	Msg  string `json:"msg"`
}

// Account holds the account data
type Account struct {
	MakerCommission  int64      `json:"makerCommission"`
	TakerCommission  int64      `json:"takerCommission"`
	BuyerCommission  int64      `json:"buyerCommission"`
	SellerCommission int64      `json:"sellerCommission"`
	CanTrade         bool       `json:"canTrade"`
	CanWithdraw      bool       `json:"canWithdraw"`
	CanDeposit       bool       `json:"canDeposit"`
	UpdateTime       types.Time `json:"updateTime"`
	AccountType      string     `json:"accountType"`
	Balances         []Balance  `json:"balances"`
	Permissions      []string   `json:"permissions"`
}

// Balance holds query order data
type Balance struct {
	Asset  currency.Code   `json:"asset"`
	Free   decimal.Decimal `json:"free"`
	Locked decimal.Decimal `json:"locked"`
}

// AccountStatusResponse holds information related to the
// User Account status information request
type AccountStatusResponse struct {
	Msg     string   `json:"msg"`
	Success bool     `json:"success"`
	Objs    []string `json:"objs,omitempty"`
}

// TradeStatus represents trade status and holds list of trade status indicator Item instances.
type TradeStatus struct {
	IsLocked           bool                                  `json:"isLocked"`
	PlannedRecoverTime uint64                                `json:"plannedRecoverTime"`
	TriggerCondition   map[string]uint64                     `json:"triggerCondition"`
	Indicators         map[string]TradingStatusIndicatorItem `json:"indicators"`
	UpdateTime         types.Time                            `json:"updateTime"`
}

// TradingStatusIndicatorItem represents Trade Status Indication
type TradingStatusIndicatorItem struct {
	IndicatorSymbol  string  `json:"i"`
	CountOfAllOrders float32 `json:"c"`
	CurrentValue     float32 `json:"v"`
	TriggerValue     float32 `json:"t"`
}

// TradeFee represents the symbol and corresponding maker and taker trading fee value.
type TradeFee struct {
	Symbol string  `json:"symbol"`
	Maker  float64 `json:"maker"`
	Taker  float64 `json:"taker"`
}

// TradeFeeList list of trading fee for different trade symbols.
type TradeFeeList struct {
	TradeFee []TradeFee `json:"tradeFee"`
	Success  bool       `json:"success,omitempty"`
}

// AssetHistory holds the asset type and translation info
type AssetHistory struct {
	Amount  float64 `json:"amount,string"` // Amount
	Asset   string  `json:"asset"`         // Asset Type eg. BHFT
	DivTime uint64  `json:"divTime"`       // DivTime
	EnInfo  string  `json:"enInfo"`        //
	TranID  uint64  `json:"tranId"`        // Transaction ID
}

// AssetDistributionHistories this endpoint to query asset distribution records,
// including for staking, referrals and airdrops etc.
type AssetDistributionHistories struct {
	Rows  []AssetHistory `json:"rows"`
	Total uint64         `json:"total"`
}

// SubAccount  holds a single sub account instance in a Binance US account.
// including the email and related information related to it.
type SubAccount struct {
	Email      string     `json:"email"`
	Status     string     `json:"status"`
	Activated  bool       `json:"activated"`
	Mobile     string     `json:"mobile"`
	GAuth      bool       `json:"gAuth"`
	CreateTime types.Time `json:"createTime"`
}

// TransferHistory a single asset transfer history between Sub accounts
type TransferHistory struct {
	From      string     `json:"from"`
	To        string     `json:"to"`
	Asset     string     `json:"asset"`
	Qty       uint64     `json:"qty,string"`
	TimeStamp types.Time `json:"time"`
}

// SubAccountTransferRequestParams contains argument variables holder used to transfer an
// asset from one account to another subaccount
type SubAccountTransferRequestParams struct {
	FromEmail  string  // Mandatory
	ToEmail    string  // Mandatory
	Asset      string  // Mandatory
	Amount     float64 // Mandatory
	RecvWindow uint64
}

// SubAccountTransferResponse represents a suabccount transfer history
// having the transaction id which is to be returned due to the transfer
type SubAccountTransferResponse struct {
	Success bool   `json:"success"`
	TxnID   uint64 `json:"txnId,string"`
}

// AssetInfo holds asset information
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

// OrderRateLimit holds rate limits type, interval, and related information of trade orders.
type OrderRateLimit struct {
	RateLimitType string `json:"rateLimitType"`
	Interval      string `json:"interval"`
	IntervalNum   uint64 `json:"intervalNum"`
	Limit         uint64 `json:"limit"`
	Count         uint64 `json:"count"`
}

// RequestParamsOrderType trade order type
type RequestParamsOrderType string

// NewOrderRequest request type
type NewOrderRequest struct {
	Symbol           currency.Pair
	Side             string
	TradeType        RequestParamsOrderType
	TimeInForce      string
	Quantity         float64
	QuoteOrderQty    float64
	Price            float64
	NewClientOrderID string
	StopPrice        float64 // Used with STOP_LOSS, STOP_LOSS_LIMIT, TAKE_PROFIT, and TAKE_PROFIT_LIMIT orders.
	IcebergQty       float64 // Used with LIMIT, STOP_LOSS_LIMIT, and TAKE_PROFIT_LIMIT to create an iceberg order.
	NewOrderRespType string
}

// NewOrderResponse represents trade order's detailed information.
type NewOrderResponse struct {
	Symbol          string     `json:"symbol"`
	OrderID         int64      `json:"orderId"`
	OrderListID     int8       `json:"orderListId"`
	ClientOrderID   string     `json:"clientOrderId"`
	TransactionTime types.Time `json:"transactTime"`
	Price           float64    `json:"price,string"`
	OrigQty         float64    `json:"origQty,string"`
	ExecutedQty     float64    `json:"executedQty,string"`
	// The cumulative amount of the quote that has been spent (with a BUY order) or received (with a SELL order).
	CumulativeQuoteQty float64 `json:"cummulativeQuoteQty,string"`
	Status             string  `json:"status"`
	TimeInForce        string  `json:"timeInForce"`
	Type               string  `json:"type"`
	Side               string  `json:"side"`
	// --
	Code int64  `json:"code"`
	Msg  string `json:"msg"`
	// --
	Fills []struct {
		Price           float64 `json:"price,string"`
		Qty             float64 `json:"qty,string"`
		Commission      float64 `json:"commission,string"`
		CommissionAsset string  `json:"commissionAsset"`
	} `json:"fills"`
}

// CommonOrder instance holds the order information common to both
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

// Order struct represents an ordinary order response.
type Order struct {
	CommonOrder
	IcebergQty        float64    `json:"icebergQty,string"`
	Time              types.Time `json:"time"`
	UpdateTime        types.Time `json:"updateTime"`
	IsWorking         bool       `json:"isWorking"`
	OrigQuoteOrderQty float64    `json:"origQuoteOrderQty,string"`
}

// OCOOrderReportItem this is used by the OCO order creating response
type OCOOrderReportItem struct {
	CommonOrder
	TransactionTime types.Time `json:"transactionTime"`
}

// OrderRequestParams this struct will be used to get a
// order and its related information
type OrderRequestParams struct {
	Symbol            string `json:"symbol"` // REQUIRED
	OrderID           uint64 `json:"orderId"`
	OrigClientOrderID string `json:"origClientOrderId"`
	recvWindow        uint64
}

// CancelOrderRequestParams this struct will be used as a parameter for
// cancel order method.
type CancelOrderRequestParams struct {
	Symbol                currency.Pair
	OrderID               string
	ClientSuppliedOrderID string
	NewClientOrderID      string
	RecvWindow            uint64
}

// GetTradesParams  request param to get the trade history
type GetTradesParams struct {
	Symbol     string     `json:"symbol"`
	OrderID    uint64     `json:"orderId"`
	StartTime  *time.Time `json:"startTime"`
	EndTime    *time.Time `json:"endTime"`
	FromID     uint64     `json:"fromId"`
	Limit      uint64     `json:"limit"`
	RecvWindow uint64     `json:"recvWindow"`
}

// Trade this struct represents a trade response.
type Trade struct {
	Symbol          string     `json:"symbol"`
	ID              uint64     `json:"id"`
	OrderID         uint64     `json:"orderId"`
	OrderListID     int64      `json:"orderListId"`
	Price           float64    `json:"price"`
	Qty             float64    `json:"qty"`
	QuoteQty        float64    `json:"quoteQty"`
	Commission      float64    `json:"commission"`
	CommissionAsset float64    `json:"commissionAsset"`
	Time            types.Time `json:"time"`
	IsBuyer         bool       `json:"isBuyer"`
	IsMaker         bool       `json:"isMaker"`
	IsBestMatch     bool       `json:"isBestMatch"`
}

// OCOOrderInputParams One-cancel-the-other order creation input Parameter
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

// GetOCOOrderRequestParams a parameter model to query specific list of OCO orders using their id
type GetOCOOrderRequestParams struct {
	OrderListID       string
	OrigClientOrderID string
}

// OrderShortResponse holds symbol Identification information of trade orders.
type OrderShortResponse struct {
	Symbol        string `json:"symbol"`
	OrderID       uint64 `json:"orderId"`
	ClientOrderID string `json:"clientOrderId"`
}

// OCOOrderResponse  this model is to be used to fetch the response of create new OCO order response
type OCOOrderResponse struct {
	OrderListID       int64                `json:"orderListId"`
	ContingencyType   string               `json:"contingencyType"`
	ListStatusType    string               `json:"listStatusType"`
	ListOrderStatus   string               `json:"listOrderStatus"`
	ListClientOrderID string               `json:"listClientOrderId"`
	TransactionTime   types.Time           `json:"transactionTime"`
	Symbol            string               `json:"symbol"`
	Orders            []OrderShortResponse `json:"orders"`
}

// OCOFullOrderResponse holds detailed OCO order information with the corresponding transaction time
type OCOFullOrderResponse struct {
	*OCOOrderResponse
	OrderReports []OCOOrderReportItem `json:"orderReports"`
}

// OCOOrdersRequestParams a parameter model to query from list of OCO orders.
type OCOOrdersRequestParams struct {
	FromID     uint64
	StartTime  time.Time
	EndTime    time.Time
	Limit      uint64
	RecvWindow uint64
}

// OCOOrdersDeleteRequestParams holds the params to delete a new order
type OCOOrdersDeleteRequestParams struct {
	Symbol            string
	OrderListID       uint64
	ListClientOrderID string
	NewClientOrderID  string
	RecvWindow        uint64
}

// OTC endpoints

// CoinPairInfo holds supported coin pair for conversion with its detailed information
type CoinPairInfo struct {
	FromCoin          string  `json:"fromCoin"`
	ToCoin            string  `json:"toCoin"`
	FromCoinMinAmount float64 `json:"fromCoinMinAmount,string"`
	FromCoinMaxAmount float64 `json:"fromCoinMaxAmount,string"`
	ToCoinMinAmount   float64 `json:"toCoinMinAmount,string"`
	ToCoinMaxAmount   float64 `json:"toCoinMaxAmount,string"`
}

// RequestQuoteParams a parameter model to query quote information
type RequestQuoteParams struct {
	FromCoin      string  `json:"fromCoin"`
	ToCoin        string  `json:"toCoin"`
	RequestCoin   string  `json:"requestCoin"`
	RequestAmount float64 `json:"requestAmount"`
}

// Quote holds quote information for from-to-coin pair
type Quote struct {
	Symbol         string     `json:"symbol"`
	Ratio          float64    `json:"ratio,string"`
	InverseRatio   float64    `json:"inverseRatio,string"`
	ValidTimestamp types.Time `json:"validTimestamp"`
	ToAmount       float64    `json:"toAmount,string"`
	FromAmount     float64    `json:"fromAmount,string"`
}

// OTCTradeOrderResponse holds OTC(over-the-counter) order identification and status information
type OTCTradeOrderResponse struct {
	OrderID     uint64     `json:"orderId,string"`
	OrderStatus string     `json:"orderStatus"`
	CreateTime  types.Time `json:"createTime"`
}

// OTCTradeOrder holds OTC(over-the-counter) orders response
type OTCTradeOrder struct {
	QuoteID      string     `json:"quoteId"`
	OrderID      uint64     `json:"orderId,string"`
	OrderStatus  string     `json:"orderStatus"`
	FromCoin     string     `json:"fromCoin"`
	FromAmount   float64    `json:"fromAmount"`
	ToCoin       string     `json:"toCoin"`
	ToAmount     float64    `json:"toAmount"`
	Ratio        float64    `json:"ratio"`
	InverseRatio float64    `json:"inverseRatio"`
	CreateTime   types.Time `json:"createTime"`
}

// OTCTradeOrderRequestParams request param for Over-the-Counter trade order params.
type OTCTradeOrderRequestParams struct {
	OrderID   string
	FromCoin  string
	ToCoin    string
	StartTime time.Time
	EndTime   time.Time
	Limit     int8
}

// Wallet Endpoints

// AssetWalletDetail represents the wallet asset information.
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
		MinConfirm              int64   `json:"minConfirm,omitempty"`
		UnLockConfirm           int64   `json:"unLockConfirm,omitempty"`
	} `json:"networkList"`
}

// AssetWalletList list of asset wallet details
type AssetWalletList []AssetWalletDetail

// WithdrawalRequestParam represents the params for the
// input parameters of Withdraw Crypto
type WithdrawalRequestParam struct {
	Coin            string  `json:"coin"`
	Network         string  `json:"network"`
	WithdrawOrderID string  `json:"withdrawOrderId"` // Client ID for withdraw
	Address         string  `json:"address"`
	AddressTag      string  `json:"addressTag"`
	Amount          float64 `json:"amount"`
	RecvWindow      uint64  `json:"recvWindow"`
}

// WithdrawalResponse holds the transaction id for a withdrawal action.
type WithdrawalResponse struct {
	ID string `json:"id"`
}

// WithdrawStatusResponse defines a withdrawal status response
type WithdrawStatusResponse struct {
	ID             string         `json:"id"`
	Amount         float64        `json:"amount,string"`
	TransactionFee float64        `json:"transactionFee,string"`
	Coin           string         `json:"coin"`
	Status         int64          `json:"status"`
	Address        string         `json:"address"`
	ApplyTime      types.DateTime `json:"applyTime"`
	Network        string         `json:"network"`
	TransferType   int64          `json:"transferType"`
}

// FiatAssetRecord asset information for fiat.
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

// FiatAssetsHistory  holds list of available fiat asset records.
type FiatAssetsHistory struct {
	AssetLogRecordList []FiatAssetRecord `json:"assetLogRecordList"`
}

// WithdrawFiatRequestParams represents the fiat withdrawal request params.
type WithdrawFiatRequestParams struct {
	PaymentChannel string
	PaymentMethod  string
	PaymentAccount string
	FiatCurrency   string
	Amount         float64
	RecvWindow     uint64
}

// FiatWithdrawalRequestParams to fetch your fiat (USD) withdrawal history.
type FiatWithdrawalRequestParams struct {
	FiatCurrency   string
	OrderID        string
	Offset         int64
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

// DepositHistory stores deposit history info.
type DepositHistory struct {
	Amount       string     `json:"amount"`
	Coin         string     `json:"coin"`
	Network      string     `json:"network"`
	Status       int64      `json:"status"`
	Address      string     `json:"address"`
	AddressTag   string     `json:"addressTag"`
	TxID         string     `json:"txId"`
	InsertTime   types.Time `json:"insertTime"`
	TransferType int64      `json:"transferType"`
	ConfirmTimes string     `json:"confirmTimes"`
}

// UserAccountStream represents the response for getting the listen key for the websocket
type UserAccountStream struct {
	ListenKey string `json:"listenKey"`
}

// WebsocketPayload defines the payload through the websocket connection
type WebsocketPayload struct {
	Method string `json:"method"`
	Params []any  `json:"params"`
	ID     int64  `json:"id"`
}

// orderbookManager defines a way of managing and maintaining synchronisation
// across connections and assets.
type orderbookManager struct {
	state map[currency.Code]map[currency.Code]map[asset.Item]*update
	sync.Mutex

	jobs chan job
}

// job defines a synchronisation job that tells a go routine to fetch an
// orderbook via the REST protocol
type job struct {
	Pair currency.Pair
}

// update holds websocket depth stream response data and update information
type update struct {
	buffer            chan *WebsocketDepthStream
	fetchingBook      bool
	initialSync       bool
	needsFetchingBook bool
	lastUpdateID      int64
}

// WebsocketDepthStream is the difference for the update depth stream
type WebsocketDepthStream struct {
	Event         string                           `json:"e"`
	Timestamp     types.Time                       `json:"E"`
	Pair          string                           `json:"s"`
	FirstUpdateID int64                            `json:"U"`
	LastUpdateID  int64                            `json:"u"`
	UpdateBids    orderbook.LevelsArrayPriceAmount `json:"b"`
	UpdateAsks    orderbook.LevelsArrayPriceAmount `json:"a"`
}

// WebsocketDepthDiffStream websocket response of depth diff stream
type WebsocketDepthDiffStream struct {
	LastUpdateID int64                            `json:"lastUpdateId"`
	Bids         orderbook.LevelsArrayPriceAmount `json:"bids"`
	Asks         orderbook.LevelsArrayPriceAmount `json:"asks"`
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

// wsAccountPosition websocket response of account position.
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
	EventTime   types.Time `json:"E"`
	LastUpdated types.Time `json:"u"`
	EventType   string     `json:"e"`
}

// wsBalanceUpdate represents the websocket response of update balance.
type wsBalanceUpdate struct {
	Stream string              `json:"stream"`
	Data   WsBalanceUpdateData `json:"data"`
}

// WsBalanceUpdateData defines websocket account balance data.
type WsBalanceUpdateData struct {
	EventTime    types.Time `json:"E"`
	ClearTime    types.Time `json:"T"`
	BalanceDelta float64    `json:"d,string"`
	Asset        string     `json:"a"`
	EventType    string     `json:"e"`
}

type wsOrderUpdate struct {
	Stream string            `json:"stream"`
	Data   WsOrderUpdateData `json:"data"`
}

// WsOrderUpdateData defines websocket account order update data
type WsOrderUpdateData struct {
	EventType                         string     `json:"e"`
	EventTime                         types.Time `json:"E"`
	Symbol                            string     `json:"s"`
	ClientOrderID                     string     `json:"c"`
	Side                              string     `json:"S"`
	OrderType                         string     `json:"o"`
	TimeInForce                       string     `json:"f"`
	Quantity                          float64    `json:"q,string"`
	Price                             float64    `json:"p,string"`
	StopPrice                         float64    `json:"P,string"`
	IcebergQuantity                   float64    `json:"F,string"`
	OrderListID                       int64      `json:"g"`
	CancelledClientOrderID            string     `json:"C"`
	CurrentExecutionType              string     `json:"x"`
	OrderStatus                       string     `json:"X"`
	RejectionReason                   string     `json:"r"`
	OrderID                           int64      `json:"i"`
	LastExecutedQuantity              float64    `json:"l,string"`
	CumulativeFilledQuantity          float64    `json:"z,string"`
	LastExecutedPrice                 float64    `json:"L,string"`
	Commission                        float64    `json:"n,string"`
	CommissionAsset                   string     `json:"N"`
	TransactionTime                   types.Time `json:"T"`
	TradeID                           int64      `json:"t"`
	Ignored                           int64      `json:"I"` // Must be ignored explicitly, otherwise it overwrites 'i'.
	IsOnOrderBook                     bool       `json:"w"`
	IsMaker                           bool       `json:"m"`
	Ignored2                          bool       `json:"M"` // See the comment for "I".
	OrderCreationTime                 types.Time `json:"O"`
	CumulativeQuoteTransactedQuantity float64    `json:"Z,string"`
	LastQuoteAssetTransactedQuantity  float64    `json:"Y,string"`
	QuoteOrderQuantity                float64    `json:"Q,string"`
}

// WsListStatus holder for websocket account listing status response including the stream information
type WsListStatus struct {
	Stream string           `json:"stream"`
	Data   WsListStatusData `json:"data"`
}

// WsListStatusData holder for websocket account listing status response.
type WsListStatusData struct {
	ListClientOrderID string     `json:"C"`
	EventTime         types.Time `json:"E"`
	ListOrderStatus   string     `json:"L"`
	Orders            []struct {
		ClientOrderID string `json:"c"`
		OrderID       int64  `json:"i"`
		Symbol        string `json:"s"`
	} `json:"O"`
	TransactionTime types.Time `json:"T"`
	ContingencyType string     `json:"c"`
	EventType       string     `json:"e"`
	OrderListID     int64      `json:"g"`
	ListStatusType  string     `json:"l"`
	RejectionReason string     `json:"r"`
	Symbol          string     `json:"s"`
}

// TradeStream holds the trade stream data
type TradeStream struct {
	EventType      string       `json:"e"`
	EventTime      types.Time   `json:"E"`
	Symbol         string       `json:"s"`
	TradeID        int64        `json:"t"`
	Price          types.Number `json:"p"`
	Quantity       types.Number `json:"q"`
	BuyerOrderID   int64        `json:"b"`
	SellerOrderID  int64        `json:"a"`
	TimeStamp      types.Time   `json:"T"`
	Maker          bool         `json:"m"`
	BestMatchPrice bool         `json:"M"`
}

// KlineStream holds the kline stream data
type KlineStream struct {
	EventType string          `json:"e"`
	EventTime types.Time      `json:"E"`
	Symbol    string          `json:"s"`
	Kline     KlineStreamData `json:"k"`
}

// KlineStreamData defines kline streaming data
type KlineStreamData struct {
	StartTime                types.Time `json:"t"`
	CloseTime                types.Time `json:"T"`
	Symbol                   string     `json:"s"`
	Interval                 string     `json:"i"`
	FirstTradeID             int64      `json:"f"`
	LastTradeID              int64      `json:"L"`
	OpenPrice                float64    `json:"o,string"`
	ClosePrice               float64    `json:"c,string"`
	HighPrice                float64    `json:"h,string"`
	LowPrice                 float64    `json:"l,string"`
	Volume                   float64    `json:"v,string"`
	NumberOfTrades           int64      `json:"n"`
	KlineClosed              bool       `json:"x"`
	Quote                    float64    `json:"q,string"`
	TakerBuyBaseAssetVolume  float64    `json:"V,string"`
	TakerBuyQuoteAssetVolume float64    `json:"Q,string"`
}

// TickerStream holds the ticker stream data
type TickerStream struct {
	EventType              string     `json:"e"`
	EventTime              types.Time `json:"E"`
	Symbol                 string     `json:"s"`
	PriceChange            float64    `json:"p,string"`
	PriceChangePercent     float64    `json:"P,string"`
	WeightedAvgPrice       float64    `json:"w,string"`
	ClosePrice             float64    `json:"x,string"`
	LastPrice              float64    `json:"c,string"`
	LastPriceQuantity      float64    `json:"Q,string"`
	BestBidPrice           float64    `json:"b,string"`
	BestBidQuantity        float64    `json:"B,string"`
	BestAskPrice           float64    `json:"a,string"`
	BestAskQuantity        float64    `json:"A,string"`
	OpenPrice              float64    `json:"o,string"`
	HighPrice              float64    `json:"h,string"`
	LowPrice               float64    `json:"l,string"`
	TotalTradedVolume      float64    `json:"v,string"`
	TotalTradedQuoteVolume float64    `json:"q,string"`
	OpenTime               types.Time `json:"O"`
	CloseTime              types.Time `json:"C"`
	FirstTradeID           int64      `json:"F"`
	LastTradeID            int64      `json:"L"`
	NumberOfTrades         int64      `json:"n"`
}

// OrderBookTickerStream  contains websocket orderbook data
type OrderBookTickerStream struct {
	LastUpdateID int64  `json:"u"`
	S            string `json:"s"`
	Symbol       currency.Pair
	BestBidPrice float64 `json:"b,string"`
	BestBidQty   float64 `json:"B,string"`
	BestAskPrice float64 `json:"a,string"`
	BestAskQty   float64 `json:"A,string"`
}

// WebsocketAggregateTradeStream aggregate trade streams push data
type WebsocketAggregateTradeStream struct {
	EventType        string     `json:"e"`
	EventTime        types.Time `json:"E"`
	Symbol           string     `json:"s"`
	AggregateTradeID int64      `json:"a"`
	Price            float64    `json:"p,string"`
	Quantity         float64    `json:"q,string"`
	FirstTradeID     int64      `json:"f"`
	LastTradeID      int64      `json:"l"`
	TradeTime        types.Time `json:"T"`
	IsMaker          bool       `json:"m"`
}

// OCBSOrderRequestParams holds parameters to retrieve OCBS orders.
type OCBSOrderRequestParams struct {
	OrderID   string
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
}

// OCBSTradeOrdersResponse holds the quantity and list of OCBS Orders.
type OCBSTradeOrdersResponse struct {
	Total     int64       `json:"total"`
	OCBSOrder []OCBSOrder `json:"dataList"`
}

// OCBSOrder holds OCBS orders details.
type OCBSOrder struct {
	QuoteID     string     `json:"quoteId"`
	OrderID     string     `json:"orderId"`
	OrderStatus string     `json:"orderStatus"`
	FromCoin    string     `json:"fromCoin"`
	FromAmount  float64    `json:"fromAmount"`
	ToCoin      string     `json:"toCoin"`
	ToAmount    float64    `json:"toAmount"`
	FeeCoin     string     `json:"feeCoin"`
	FeeAmount   float64    `json:"feeAmount"`
	Ratio       float64    `json:"ratio"`
	CreateTime  types.Time `json:"createTime"`
}

// ServerTime holds the exchange server time
type ServerTime struct {
	Timestamp types.Time `json:"serverTime"`
}

// SubUserToBTCAssets holds the number of BTC assets and the corresponding sub user email.
type SubUserToBTCAssets struct {
	Email      string `json:"email"`
	TotalAsset int64  `json:"totalAsset"`
}

// SpotUSDMasterAccounts holds the USD assets of a sub user.
type SpotUSDMasterAccounts struct {
	TotalCount                    int64                `json:"totalCount"`
	MasterAccountTotalAsset       int64                `json:"masterAccountTotalAsset"`
	SpotSubUserAssetBTCVolumeList []SubUserToBTCAssets `json:"spotSubUserAssetBtcVoList"`
}

// SubAccountStatus represents single sub accounts status information.
type SubAccountStatus struct {
	Email            string     `json:"email"`
	InsertTime       types.Time `json:"insertTime"`
	Mobile           string     `json:"mobile"`
	IsUserActive     bool       `json:"isUserActive"`
	IsMarginEnabled  bool       `json:"isMarginEnabled"`
	IsSubUserEnabled bool       `json:"isSubUserEnabled"`
	IsFutureEnabled  bool       `json:"isFutureEnabled"`
}

// SubAccountDepositAddressRequestParams holds query parameters for Sub-account deposit addresses.
type SubAccountDepositAddressRequestParams struct {
	Email   string        // [Required] Sub-account email
	Coin    currency.Code // [Required]
	Network string        // Network (If empty, returns the default network)
}

// SubAccountDepositAddress holds sub-accounts deposit address information
type SubAccountDepositAddress struct {
	Coin    string `json:"coin"`
	Address string `json:"address"`
	Tag     string `json:"tag"`
	URL     string `json:"url"`
}

// SubAccountDepositItem holds the sub-account deposit information
type SubAccountDepositItem struct {
	Amount        string     `json:"amount"`
	Coin          string     `json:"coin"`
	Network       string     `json:"network"`
	Status        int64      `json:"status"`
	Address       string     `json:"address"`
	AddressTag    string     `json:"addressTag"`
	TransactionID string     `json:"txId"`
	InsertTime    types.Time `json:"insertTime"`
	TransferType  int64      `json:"transferType"`
	ConfirmTimes  string     `json:"confirmTimes"`
}

// ReferralRewardHistoryResponse holds reward history response
type ReferralRewardHistoryResponse struct {
	Total int64                    `json:"total"`
	Rows  []ReferralWithdrawalItem `json:"rows"`
}

// ReferralWithdrawalItem holds reward history item
type ReferralWithdrawalItem struct {
	UserID          int64      `json:"userId"`
	RewardAmount    string     `json:"rewardAmount"`
	ReceiveDateTime types.Time `json:"receiveDateTime"`
	RewardType      string     `json:"rewardType"`
}

// SpotAssetsSnapshotResponse represents spot asset types snapshot information.
type SpotAssetsSnapshotResponse struct {
	Code        int64    `json:"code"`
	Msg         string   `json:"msg"`
	SnapshotVos []string `json:"snapshotVos"`
}
