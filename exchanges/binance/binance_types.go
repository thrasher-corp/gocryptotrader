package binance

import (
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
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

// withdrawalFees the large list of predefined withdrawal fees. Prone to change.
var withdrawalFees = map[asset.Item]map[currency.Code]fee.Transfer{
	asset.Spot: {
		currency.BNB:     {Withdrawal: 0.13},
		currency.BTC:     {Withdrawal: 0.0005},
		currency.NEO:     {Withdrawal: 0},
		currency.ETH:     {Withdrawal: 0.01},
		currency.LTC:     {Withdrawal: 0.001},
		currency.QTUM:    {Withdrawal: 0.01},
		currency.EOS:     {Withdrawal: 0.1},
		currency.SNT:     {Withdrawal: 35},
		currency.BNT:     {Withdrawal: 1},
		currency.GAS:     {Withdrawal: 0},
		currency.BCC:     {Withdrawal: 0.001},
		currency.BTM:     {Withdrawal: 5},
		currency.USDT:    {Withdrawal: 3.4},
		currency.HCC:     {Withdrawal: 0.0005},
		currency.OAX:     {Withdrawal: 6.5},
		currency.DNT:     {Withdrawal: 54},
		currency.MCO:     {Withdrawal: 0.31},
		currency.ICN:     {Withdrawal: 3.5},
		currency.ZRX:     {Withdrawal: 1.9},
		currency.OMG:     {Withdrawal: 0.4},
		currency.WTC:     {Withdrawal: 0.5},
		currency.LRC:     {Withdrawal: 12.3},
		currency.LLT:     {Withdrawal: 67.8},
		currency.YOYO:    {Withdrawal: 1},
		currency.TRX:     {Withdrawal: 1},
		currency.STRAT:   {Withdrawal: 0.1},
		currency.SNGLS:   {Withdrawal: 54},
		currency.BQX:     {Withdrawal: 3.9},
		currency.KNC:     {Withdrawal: 3.5},
		currency.SNM:     {Withdrawal: 25},
		currency.FUN:     {Withdrawal: 86},
		currency.LINK:    {Withdrawal: 4},
		currency.XVG:     {Withdrawal: 0.1},
		currency.CTR:     {Withdrawal: 35},
		currency.SALT:    {Withdrawal: 2.3},
		currency.MDA:     {Withdrawal: 2.3},
		currency.IOTA:    {Withdrawal: 0.5},
		currency.SUB:     {Withdrawal: 11.4},
		currency.ETC:     {Withdrawal: 0.01},
		currency.MTL:     {Withdrawal: 2},
		currency.MTH:     {Withdrawal: 45},
		currency.ENG:     {Withdrawal: 2.2},
		currency.AST:     {Withdrawal: 14.4},
		currency.DASH:    {Withdrawal: 0.002},
		currency.BTG:     {Withdrawal: 0.001},
		currency.EVX:     {Withdrawal: 2.8},
		currency.REQ:     {Withdrawal: 29.9},
		currency.VIB:     {Withdrawal: 30},
		currency.POWR:    {Withdrawal: 8.2},
		currency.ARK:     {Withdrawal: 0.2},
		currency.XRP:     {Withdrawal: 0.25},
		currency.MOD:     {Withdrawal: 2},
		currency.ENJ:     {Withdrawal: 26},
		currency.STORJ:   {Withdrawal: 5.1},
		currency.KMD:     {Withdrawal: 0.002},
		currency.RCN:     {Withdrawal: 47},
		currency.NULS:    {Withdrawal: 0.01},
		currency.RDN:     {Withdrawal: 2.5},
		currency.XMR:     {Withdrawal: 0.04},
		currency.DLT:     {Withdrawal: 19.8},
		currency.AMB:     {Withdrawal: 8.9},
		currency.BAT:     {Withdrawal: 8},
		currency.ZEC:     {Withdrawal: 0.005},
		currency.BCPT:    {Withdrawal: 14.5},
		currency.ARN:     {Withdrawal: 3},
		currency.GVT:     {Withdrawal: 0.13},
		currency.CDT:     {Withdrawal: 81},
		currency.GXS:     {Withdrawal: 0.3},
		currency.POE:     {Withdrawal: 134},
		currency.QSP:     {Withdrawal: 36},
		currency.BTS:     {Withdrawal: 1},
		currency.XZC:     {Withdrawal: 0.02},
		currency.LSK:     {Withdrawal: 0.1},
		currency.TNT:     {Withdrawal: 47},
		currency.FUEL:    {Withdrawal: 79},
		currency.MANA:    {Withdrawal: 18},
		currency.BCD:     {Withdrawal: 0.01},
		currency.DGD:     {Withdrawal: 0.04},
		currency.ADX:     {Withdrawal: 6.3},
		currency.ADA:     {Withdrawal: 1},
		currency.PPT:     {Withdrawal: 0.41},
		currency.CMT:     {Withdrawal: 12},
		currency.XLM:     {Withdrawal: 0.01},
		currency.CND:     {Withdrawal: 58},
		currency.LEND:    {Withdrawal: 84},
		currency.WABI:    {Withdrawal: 6.6},
		currency.SBTC:    {Withdrawal: 0.0005},
		currency.BCX:     {Withdrawal: 0.5},
		currency.WAVES:   {Withdrawal: 0.002},
		currency.TNB:     {Withdrawal: 139},
		currency.GTO:     {Withdrawal: 20},
		currency.ICX:     {Withdrawal: 0.02},
		currency.OST:     {Withdrawal: 32},
		currency.ELF:     {Withdrawal: 3.9},
		currency.AION:    {Withdrawal: 3.2},
		currency.CVC:     {Withdrawal: 10.9},
		currency.REP:     {Withdrawal: 0.2},
		currency.GNT:     {Withdrawal: 8.9},
		currency.DATA:    {Withdrawal: 37},
		currency.ETF:     {Withdrawal: 1},
		currency.BRD:     {Withdrawal: 3.8},
		currency.NEBL:    {Withdrawal: 0.01},
		currency.VIBE:    {Withdrawal: 17.3},
		currency.LUN:     {Withdrawal: 0.36},
		currency.CHAT:    {Withdrawal: 60.7},
		currency.RLC:     {Withdrawal: 3.4},
		currency.INS:     {Withdrawal: 3.5},
		currency.IOST:    {Withdrawal: 105.6},
		currency.STEEM:   {Withdrawal: 0.01},
		currency.NANO:    {Withdrawal: 0.01},
		currency.AE:      {Withdrawal: 1.3},
		currency.VIA:     {Withdrawal: 0.01},
		currency.BLZ:     {Withdrawal: 10.3},
		currency.SYS:     {Withdrawal: 1},
		currency.NCASH:   {Withdrawal: 247.6},
		currency.POA:     {Withdrawal: 0.01},
		currency.ONT:     {Withdrawal: 1},
		currency.ZIL:     {Withdrawal: 37.2},
		currency.STORM:   {Withdrawal: 152},
		currency.XEM:     {Withdrawal: 4},
		currency.WAN:     {Withdrawal: 0.1},
		currency.WPR:     {Withdrawal: 43.4},
		currency.QLC:     {Withdrawal: 1},
		currency.GRS:     {Withdrawal: 0.2},
		currency.CLOAK:   {Withdrawal: 0.02},
		currency.LOOM:    {Withdrawal: 11.9},
		currency.BCN:     {Withdrawal: 1},
		currency.TUSD:    {Withdrawal: 1.35},
		currency.ZEN:     {Withdrawal: 0.002},
		currency.SKY:     {Withdrawal: 0.01},
		currency.THETA:   {Withdrawal: 24},
		currency.IOTX:    {Withdrawal: 90.5},
		currency.QKC:     {Withdrawal: 24.6},
		currency.AGI:     {Withdrawal: 29.81},
		currency.NXS:     {Withdrawal: 0.02},
		currency.SC:      {Withdrawal: 0.1},
		currency.EON:     {Withdrawal: 10},
		currency.NPXS:    {Withdrawal: 897},
		currency.KEY:     {Withdrawal: 223},
		currency.NAS:     {Withdrawal: 0.1},
		currency.ADD:     {Withdrawal: 100},
		currency.MEETONE: {Withdrawal: 300},
		currency.ATD:     {Withdrawal: 100},
		currency.MFT:     {Withdrawal: 175},
		currency.EOP:     {Withdrawal: 5},
		currency.DENT:    {Withdrawal: 596},
		currency.IQ:      {Withdrawal: 50},
		currency.ARDR:    {Withdrawal: 2},
		currency.HOT:     {Withdrawal: 1210},
		currency.VET:     {Withdrawal: 100},
		currency.DOCK:    {Withdrawal: 68},
		currency.POLY:    {Withdrawal: 7},
		currency.VTHO:    {Withdrawal: 21},
		currency.ONG:     {Withdrawal: 0.1},
		currency.PHX:     {Withdrawal: 1},
		currency.HC:      {Withdrawal: 0.005},
		currency.GO:      {Withdrawal: 0.01},
		currency.PAX:     {Withdrawal: 1.4},
		currency.EDO:     {Withdrawal: 1.3},
		currency.WINGS:   {Withdrawal: 8.9},
		currency.NAV:     {Withdrawal: 0.2},
		currency.TRIG:    {Withdrawal: 49.1},
		currency.APPC:    {Withdrawal: 12.4},
		currency.PIVX:    {Withdrawal: 0.02},
	},
}

// WithdrawResponse contains status of withdrawal request
type WithdrawResponse struct {
	Success bool   `json:"success"`
	Msg     string `json:"msg"`
	ID      string `json:"id"`
}

// WithdrawStatusResponse defines a withdrawal status response
type WithdrawStatusResponse struct {
	Amount         float64 `json:"amount"`
	TransactionFee float64 `json:"transactionFee"`
	Address        string  `json:"address"`
	TxID           string  `json:"txId"`
	ID             string  `json:"id"`
	Asset          string  `json:"asset"`
	ApplyTime      int64   `json:"applyTime"`
	Status         int64   `json:"status"`
	Network        string  `json:"network"`
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
	buffer       chan *WebsocketDepthStream
	fetchingBook bool
	initialSync  bool
	lastUpdateID int64
}

// job defines a synchonisation job that tells a go routine to fetch an
// orderbook via the REST protocol
type job struct {
	Pair currency.Pair
}
