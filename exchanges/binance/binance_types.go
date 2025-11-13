package binance

import (
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/types"
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

type filterType string

const (
	priceFilter              filterType = "PRICE_FILTER"
	lotSizeFilter            filterType = "LOT_SIZE"
	icebergPartsFilter       filterType = "ICEBERG_PARTS"
	marketLotSizeFilter      filterType = "MARKET_LOT_SIZE"
	trailingDeltaFilter      filterType = "TRAILING_DELTA"
	percentPriceFilter       filterType = "PERCENT_PRICE"
	percentPriceBySizeFilter filterType = "PERCENT_PRICE_BY_SIDE"
	notionalFilter           filterType = "NOTIONAL"
	maxNumOrdersFilter       filterType = "MAX_NUM_ORDERS"
	maxNumAlgoOrdersFilter   filterType = "MAX_NUM_ALGO_ORDERS"
)

// ExchangeInfo holds the full exchange information type
type ExchangeInfo struct {
	Code       int        `json:"code"`
	Msg        string     `json:"msg"`
	Timezone   string     `json:"timezone"`
	ServerTime types.Time `json:"serverTime"`
	RateLimits []*struct {
		RateLimitType string `json:"rateLimitType"`
		Interval      string `json:"interval"`
		Limit         int    `json:"limit"`
	} `json:"rateLimits"`
	ExchangeFilters any `json:"exchangeFilters"`
	Symbols         []*struct {
		Symbol                     string        `json:"symbol"`
		Status                     string        `json:"status"`
		BaseAsset                  string        `json:"baseAsset"`
		BaseAssetPrecision         int           `json:"baseAssetPrecision"`
		QuoteAsset                 string        `json:"quoteAsset"`
		QuotePrecision             int           `json:"quotePrecision"`
		OrderTypes                 []string      `json:"orderTypes"`
		IcebergAllowed             bool          `json:"icebergAllowed"`
		OCOAllowed                 bool          `json:"ocoAllowed"`
		QuoteOrderQtyMarketAllowed bool          `json:"quoteOrderQtyMarketAllowed"`
		IsSpotTradingAllowed       bool          `json:"isSpotTradingAllowed"`
		IsMarginTradingAllowed     bool          `json:"isMarginTradingAllowed"`
		Filters                    []*filterData `json:"filters"`
		Permissions                []string      `json:"permissions"`
		PermissionSets             [][]string    `json:"permissionSets"`
	} `json:"symbols"`
}

type filterData struct {
	FilterType          filterType `json:"filterType"`
	MinPrice            float64    `json:"minPrice,string"`
	MaxPrice            float64    `json:"maxPrice,string"`
	TickSize            float64    `json:"tickSize,string"`
	MultiplierUp        float64    `json:"multiplierUp,string"`
	MultiplierDown      float64    `json:"multiplierDown,string"`
	AvgPriceMinutes     int64      `json:"avgPriceMins"`
	MinQty              float64    `json:"minQty,string"`
	MaxQty              float64    `json:"maxQty,string"`
	StepSize            float64    `json:"stepSize,string"`
	MinNotional         float64    `json:"minNotional,string"`
	ApplyToMarket       bool       `json:"applyToMarket"`
	Limit               int64      `json:"limit"`
	MaxNumAlgoOrders    int64      `json:"maxNumAlgoOrders"`
	MaxNumIcebergOrders int64      `json:"maxNumIcebergOrders"`
	MaxNumOrders        int64      `json:"maxNumOrders"`
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

// OrderBookResponse is resp data from orderbook endpoint
type OrderBookResponse struct {
	Code         int64                            `json:"code"`
	Msg          string                           `json:"msg"`
	LastUpdateID int64                            `json:"lastUpdateId"`
	Timestamp    types.Time                       `json:"T"`
	Bids         orderbook.LevelsArrayPriceAmount `json:"bids"`
	Asks         orderbook.LevelsArrayPriceAmount `json:"asks"`
}

// DepthUpdateParams is used as an embedded type for WebsocketDepthStream
type DepthUpdateParams []struct {
	PriceLevel float64
	Quantity   float64
	ignore     []any
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

// RecentTradeRequestParams represents Klines request data.
type RecentTradeRequestParams struct {
	Symbol currency.Pair `json:"symbol"` // Required field. example LTCBTC, BTCUSDT
	Limit  int           `json:"limit"`  // Default 500; max 500.
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
	IsBuyerMaker   bool         `json:"m"`
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
	StartTime                types.Time   `json:"t"`
	CloseTime                types.Time   `json:"T"`
	Symbol                   string       `json:"s"`
	Interval                 string       `json:"i"`
	FirstTradeID             int64        `json:"f"`
	LastTradeID              int64        `json:"L"`
	OpenPrice                types.Number `json:"o"`
	ClosePrice               types.Number `json:"c"`
	HighPrice                types.Number `json:"h"`
	LowPrice                 types.Number `json:"l"`
	Volume                   types.Number `json:"v"`
	NumberOfTrades           int64        `json:"n"`
	KlineClosed              bool         `json:"x"`
	Quote                    types.Number `json:"q"`
	TakerBuyBaseAssetVolume  types.Number `json:"V"`
	TakerBuyQuoteAssetVolume types.Number `json:"Q"`
}

// TickerStream holds the ticker stream data
type TickerStream struct {
	EventType              string       `json:"e"`
	EventTime              types.Time   `json:"E"`
	Symbol                 string       `json:"s"`
	PriceChange            types.Number `json:"p"`
	PriceChangePercent     types.Number `json:"P"`
	WeightedAvgPrice       types.Number `json:"w"`
	ClosePrice             types.Number `json:"x"`
	LastPrice              types.Number `json:"c"`
	LastPriceQuantity      types.Number `json:"Q"`
	BestBidPrice           types.Number `json:"b"`
	BestBidQuantity        types.Number `json:"B"`
	BestAskPrice           types.Number `json:"a"`
	BestAskQuantity        types.Number `json:"A"`
	OpenPrice              types.Number `json:"o"`
	HighPrice              types.Number `json:"h"`
	LowPrice               types.Number `json:"l"`
	TotalTradedVolume      types.Number `json:"v"`
	TotalTradedQuoteVolume types.Number `json:"q"`
	OpenTime               types.Time   `json:"O"`
	CloseTime              types.Time   `json:"C"`
	FirstTradeID           int64        `json:"F"`
	LastTradeID            int64        `json:"L"`
	NumberOfTrades         int64        `json:"n"`
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
	IsBuyerMaker   bool       `json:"m"`
	BestMatchPrice bool       `json:"M"`
}

// IndexMarkPrice stores data for index and mark prices
type IndexMarkPrice struct {
	Symbol               string       `json:"symbol"`
	Pair                 string       `json:"pair"`
	MarkPrice            types.Number `json:"markPrice"`
	IndexPrice           types.Number `json:"indexPrice"`
	EstimatedSettlePrice types.Number `json:"estimatedSettlePrice"`
	LastFundingRate      types.Number `json:"lastFundingRate"`
	NextFundingTime      types.Time   `json:"nextFundingTime"`
	Time                 types.Time   `json:"time"`
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
	TradeCount               int64
	TakerBuyAssetVolume      types.Number
	TakerBuyQuoteAssetVolume types.Number
}

// UnmarshalJSON unmarshals JSON data into a CandleStick struct
func (c *CandleStick) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[11]any{
		&c.OpenTime,
		&c.Open,
		&c.High,
		&c.Low,
		&c.Close,
		&c.Volume,
		&c.CloseTime,
		&c.QuoteAssetVolume,
		&c.TradeCount,
		&c.TakerBuyAssetVolume,
		&c.TakerBuyQuoteAssetVolume,
	})
}

// AveragePrice holds current average symbol price
type AveragePrice struct {
	Mins  int64   `json:"mins"`
	Price float64 `json:"price,string"`
}

// PriceChangeStats contains statistics for the last 24 hours trade
type PriceChangeStats struct {
	Symbol             string       `json:"symbol"`
	PriceChange        types.Number `json:"priceChange"`
	PriceChangePercent types.Number `json:"priceChangePercent"`
	WeightedAvgPrice   types.Number `json:"weightedAvgPrice"`
	PrevClosePrice     types.Number `json:"prevClosePrice"`
	LastPrice          types.Number `json:"lastPrice"`
	LastQty            types.Number `json:"lastQty"`
	BidPrice           types.Number `json:"bidPrice"`
	AskPrice           types.Number `json:"askPrice"`
	BidQuantity        types.Number `json:"bidQty"`
	AskQuantity        types.Number `json:"askQty"`
	OpenPrice          types.Number `json:"openPrice"`
	HighPrice          types.Number `json:"highPrice"`
	LowPrice           types.Number `json:"lowPrice"`
	Volume             types.Number `json:"volume"`
	QuoteVolume        types.Number `json:"quoteVolume"`
	OpenTime           types.Time   `json:"openTime"`
	CloseTime          types.Time   `json:"closeTime"`
	FirstID            int64        `json:"firstId"`
	LastID             int64        `json:"lastId"`
	Count              int64        `json:"count"`
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
	TimeInForce string
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
	Code            int        `json:"code"`
	Msg             string     `json:"msg"`
	Symbol          string     `json:"symbol"`
	OrderID         int64      `json:"orderId"`
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
	Code                int        `json:"code"`
	Msg                 string     `json:"msg"`
	Symbol              string     `json:"symbol"`
	OrderID             int64      `json:"orderId"`
	ClientOrderID       string     `json:"clientOrderId"`
	Price               float64    `json:"price,string"`
	OrigQty             float64    `json:"origQty,string"`
	ExecutedQty         float64    `json:"executedQty,string"`
	Status              string     `json:"status"`
	TimeInForce         string     `json:"timeInForce"`
	Type                string     `json:"type"`
	Side                string     `json:"side"`
	StopPrice           float64    `json:"stopPrice,string"`
	IcebergQty          float64    `json:"icebergQty,string"`
	Time                types.Time `json:"time"`
	IsWorking           bool       `json:"isWorking"`
	CummulativeQuoteQty float64    `json:"cummulativeQuoteQty,string"`
	OrderListID         int64      `json:"orderListId"`
	OrigQuoteOrderQty   float64    `json:"origQuoteOrderQty,string"`
	UpdateTime          types.Time `json:"updateTime"`
}

// Balance holds query order data
type Balance struct {
	Asset  currency.Code   `json:"asset"`
	Free   decimal.Decimal `json:"free"`
	Locked decimal.Decimal `json:"locked"`
}

// Account holds the account data
type Account struct {
	MakerCommission  int        `json:"makerCommission"`
	TakerCommission  int        `json:"takerCommission"`
	BuyerCommission  int        `json:"buyerCommission"`
	SellerCommission int        `json:"sellerCommission"`
	CanTrade         bool       `json:"canTrade"`
	CanWithdraw      bool       `json:"canWithdraw"`
	CanDeposit       bool       `json:"canDeposit"`
	UpdateTime       types.Time `json:"updateTime"`
	Balances         []Balance  `json:"balances"`
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
	Asset    currency.Code `json:"asset"`
	Borrowed float64       `json:"borrowed,string"`
	Free     float64       `json:"free,string"`
	Interest float64       `json:"interest,string"`
	Locked   float64       `json:"locked,string"`
	NetAsset float64       `json:"netAsset,string"`
}

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
	Limit     uint64        // Default 500; max 500.
	StartTime time.Time
	EndTime   time.Time
}

// WithdrawalFees the large list of predefined withdrawal fees
// Prone to change
var WithdrawalFees = map[currency.Code]float64{
	currency.BNB:     0.13,
	currency.BTC:     0.0005,
	currency.NEO:     0,
	currency.ETH:     0.01,
	currency.LTC:     0.001,
	currency.QTUM:    0.01,
	currency.EOS:     0.1,
	currency.SNT:     35,
	currency.BNT:     1,
	currency.GAS:     0,
	currency.BCC:     0.001,
	currency.BTM:     5,
	currency.USDT:    3.4,
	currency.HCC:     0.0005,
	currency.OAX:     6.5,
	currency.DNT:     54,
	currency.MCO:     0.31,
	currency.ICN:     3.5,
	currency.ZRX:     1.9,
	currency.OMG:     0.4,
	currency.WTC:     0.5,
	currency.LRC:     12.3,
	currency.LLT:     67.8,
	currency.YOYO:    1,
	currency.TRX:     1,
	currency.STRAT:   0.1, //nolint:misspell // Not a misspelling
	currency.SNGLS:   54,
	currency.BQX:     3.9,
	currency.KNC:     3.5,
	currency.SNM:     25,
	currency.FUN:     86,
	currency.LINK:    4,
	currency.XVG:     0.1,
	currency.CTR:     35,
	currency.SALT:    2.3,
	currency.MDA:     2.3,
	currency.IOTA:    0.5,
	currency.SUB:     11.4,
	currency.ETC:     0.01,
	currency.MTL:     2,
	currency.MTH:     45,
	currency.ENG:     2.2,
	currency.AST:     14.4,
	currency.DASH:    0.002,
	currency.BTG:     0.001,
	currency.EVX:     2.8,
	currency.REQ:     29.9,
	currency.VIB:     30,
	currency.POWR:    8.2,
	currency.ARK:     0.2,
	currency.XRP:     0.25,
	currency.MOD:     2,
	currency.ENJ:     26,
	currency.STORJ:   5.1,
	currency.KMD:     0.002,
	currency.RCN:     47,
	currency.NULS:    0.01,
	currency.RDN:     2.5,
	currency.XMR:     0.04,
	currency.DLT:     19.8,
	currency.AMB:     8.9,
	currency.BAT:     8,
	currency.ZEC:     0.005,
	currency.BCPT:    14.5,
	currency.ARN:     3,
	currency.GVT:     0.13,
	currency.CDT:     81,
	currency.GXS:     0.3,
	currency.POE:     134,
	currency.QSP:     36,
	currency.BTS:     1,
	currency.XZC:     0.02,
	currency.LSK:     0.1,
	currency.TNT:     47,
	currency.FUEL:    79,
	currency.MANA:    18,
	currency.BCD:     0.01,
	currency.DGD:     0.04,
	currency.ADX:     6.3,
	currency.ADA:     1,
	currency.PPT:     0.41,
	currency.CMT:     12,
	currency.XLM:     0.01,
	currency.CND:     58,
	currency.LEND:    84,
	currency.WABI:    6.6,
	currency.SBTC:    0.0005,
	currency.BCX:     0.5,
	currency.WAVES:   0.002,
	currency.TNB:     139,
	currency.GTO:     20,
	currency.ICX:     0.02,
	currency.OST:     32,
	currency.ELF:     3.9,
	currency.AION:    3.2,
	currency.CVC:     10.9,
	currency.REP:     0.2,
	currency.GNT:     8.9,
	currency.DATA:    37,
	currency.ETF:     1,
	currency.BRD:     3.8,
	currency.NEBL:    0.01,
	currency.VIBE:    17.3,
	currency.LUN:     0.36,
	currency.CHAT:    60.7,
	currency.RLC:     3.4,
	currency.INS:     3.5,
	currency.IOST:    105.6,
	currency.STEEM:   0.01,
	currency.NANO:    0.01,
	currency.AE:      1.3,
	currency.VIA:     0.01,
	currency.BLZ:     10.3,
	currency.SYS:     1,
	currency.NCASH:   247.6,
	currency.POA:     0.01,
	currency.ONT:     1,
	currency.ZIL:     37.2,
	currency.STORM:   152,
	currency.XEM:     4,
	currency.WAN:     0.1,
	currency.WPR:     43.4,
	currency.QLC:     1,
	currency.GRS:     0.2,
	currency.CLOAK:   0.02,
	currency.LOOM:    11.9,
	currency.BCN:     1,
	currency.TUSD:    1.35,
	currency.ZEN:     0.002,
	currency.SKY:     0.01,
	currency.THETA:   24,
	currency.IOTX:    90.5,
	currency.QKC:     24.6,
	currency.AGI:     29.81,
	currency.NXS:     0.02,
	currency.SC:      0.1,
	currency.EON:     10,
	currency.NPXS:    897,
	currency.KEY:     223,
	currency.NAS:     0.1,
	currency.ADD:     100,
	currency.MEETONE: 300,
	currency.ATD:     100,
	currency.MFT:     175,
	currency.EOP:     5,
	currency.DENT:    596,
	currency.IQ:      50,
	currency.ARDR:    2,
	currency.HOT:     1210,
	currency.VET:     100,
	currency.DOCK:    68,
	currency.POLY:    7,
	currency.VTHO:    21,
	currency.ONG:     0.1,
	currency.PHX:     1,
	currency.HC:      0.005,
	currency.GO:      0.01,
	currency.PAX:     1.4,
	currency.EDO:     1.3,
	currency.WINGS:   8.9,
	currency.NAV:     0.2,
	currency.TRIG:    49.1,
	currency.APPC:    12.4,
	currency.PIVX:    0.02,
}

// DepositHistory stores deposit history info
type DepositHistory struct {
	Amount        float64    `json:"amount,string"`
	Coin          string     `json:"coin"`
	Network       string     `json:"network"`
	Status        uint8      `json:"status"`
	Address       string     `json:"address"`
	AddressTag    string     `json:"adressTag"`
	TransactionID string     `json:"txId"`
	InsertTime    types.Time `json:"insertTime"`
	TransferType  uint8      `json:"transferType"`
	ConfirmTimes  string     `json:"confirmTimes"`
}

// WithdrawResponse contains status of withdrawal request
type WithdrawResponse struct {
	ID string `json:"id"`
}

// WithdrawStatusResponse defines a withdrawal status response
type WithdrawStatusResponse struct {
	Address         string     `json:"address"`
	Amount          float64    `json:"amount,string"`
	ApplyTime       types.Time `json:"applyTime"`
	Coin            string     `json:"coin"`
	ID              string     `json:"id"`
	WithdrawOrderID string     `json:"withdrawOrderId"`
	Network         string     `json:"network"`
	TransferType    uint8      `json:"transferType"`
	Status          int64      `json:"status"`
	TransactionFee  float64    `json:"transactionFee,string"`
	TransactionID   string     `json:"txId"`
	ConfirmNumber   int64      `json:"confirmNo"`
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

// WsBalanceUpdateData defines websocket account balance data
type WsBalanceUpdateData struct {
	EventTime    types.Time `json:"E"`
	ClearTime    types.Time `json:"T"`
	BalanceDelta float64    `json:"d,string"`
	Asset        string     `json:"a"`
	EventType    string     `json:"e"`
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
	WorkingTime                       types.Time `json:"W"`
	CumulativeQuoteTransactedQuantity float64    `json:"Z,string"`
	LastQuoteAssetTransactedQuantity  float64    `json:"Y,string"`
	QuoteOrderQuantity                float64    `json:"Q,string"`
}

// WsListStatusData defines websocket account listing status data
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

// WsPayload defines the payload through the websocket connection
type WsPayload struct {
	Method string   `json:"method"`
	Params []string `json:"params"`
	ID     string   `json:"id"`
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

// job defines a synchronisation job that tells a go routine to fetch an
// orderbook via the REST protocol
type job struct {
	Pair currency.Pair
}

// UserMarginInterestHistoryResponse user margin interest history response
type UserMarginInterestHistoryResponse struct {
	Rows  []UserMarginInterestHistory `json:"rows"`
	Total int64                       `json:"total"`
}

// UserMarginInterestHistory user margin interest history row
type UserMarginInterestHistory struct {
	TxID                int64      `json:"txId"`
	InterestAccruedTime types.Time `json:"interestAccuredTime"` // typo in docs, cannot verify due to API restrictions
	Asset               string     `json:"asset"`
	RawAsset            string     `json:"rawAsset"`
	Principal           float64    `json:"principal,string"`
	Interest            float64    `json:"interest,string"`
	InterestRate        float64    `json:"interestRate,string"`
	Type                string     `json:"type"`
	IsolatedSymbol      string     `json:"isolatedSymbol"`
}

// CryptoLoansIncomeHistory stores crypto loan income history data
type CryptoLoansIncomeHistory struct {
	Asset         currency.Code `json:"asset"`
	Type          string        `json:"type"`
	Amount        float64       `json:"amount,string"`
	TransactionID int64         `json:"tranId"`
}

// CryptoLoanBorrow stores crypto loan borrow data
type CryptoLoanBorrow struct {
	LoanCoin           currency.Code `json:"loanCoin"`
	Amount             float64       `json:"amount,string"`
	CollateralCoin     currency.Code `json:"collateralCoin"`
	CollateralAmount   float64       `json:"collateralAmount,string"`
	HourlyInterestRate float64       `json:"hourlyInterestRate,string"`
	OrderID            int64         `json:"orderId,string"`
}

// LoanBorrowHistoryItem stores loan borrow history item data
type LoanBorrowHistoryItem struct {
	OrderID                 int64         `json:"orderId"`
	LoanCoin                currency.Code `json:"loanCoin"`
	InitialLoanAmount       float64       `json:"initialLoanAmount,string"`
	HourlyInterestRate      float64       `json:"hourlyInterestRate,string"`
	LoanTerm                int64         `json:"loanTerm,string"`
	CollateralCoin          currency.Code `json:"collateralCoin"`
	InitialCollateralAmount float64       `json:"initialCollateralAmount,string"`
	BorrowTime              types.Time    `json:"borrowTime"`
	Status                  string        `json:"status"`
}

// LoanBorrowHistory stores loan borrow history data
type LoanBorrowHistory struct {
	Rows  []LoanBorrowHistoryItem `json:"rows"`
	Total int64                   `json:"total"`
}

// CryptoLoanOngoingOrderItem stores crypto loan ongoing order item data
type CryptoLoanOngoingOrderItem struct {
	OrderID          int64         `json:"orderId"`
	LoanCoin         currency.Code `json:"loanCoin"`
	TotalDebt        float64       `json:"totalDebt,string"`
	ResidualInterest float64       `json:"residualInterest,string"`
	CollateralCoin   currency.Code `json:"collateralCoin"`
	CollateralAmount float64       `json:"collateralAmount,string"`
	CurrentLTV       float64       `json:"currentLTV,string"`
	ExpirationTime   types.Time    `json:"expirationTime"`
}

// CryptoLoanOngoingOrder stores crypto loan ongoing order data
type CryptoLoanOngoingOrder struct {
	Rows  []CryptoLoanOngoingOrderItem `json:"rows"`
	Total int64                        `json:"total"`
}

// CryptoLoanRepay stores crypto loan repayment data
type CryptoLoanRepay struct {
	LoanCoin            currency.Code `json:"loanCoin"`
	RemainingPrincipal  float64       `json:"remainingPrincipal,string"`
	RemainingInterest   float64       `json:"remainingInterest,string"`
	CollateralCoin      currency.Code `json:"collateralCoin"`
	RemainingCollateral float64       `json:"remainingCollateral,string"`
	CurrentLTV          float64       `json:"currentLTV,string"`
	RepayStatus         string        `json:"repayStatus"`
}

// CryptoLoanRepayHistoryItem stores crypto loan repayment history item data
type CryptoLoanRepayHistoryItem struct {
	LoanCoin         currency.Code `json:"loanCoin"`
	RepayAmount      float64       `json:"repayAmount,string"`
	CollateralCoin   currency.Code `json:"collateralCoin"`
	CollateralUsed   float64       `json:"collateralUsed,string"`
	CollateralReturn float64       `json:"collateralReturn,string"`
	RepayType        string        `json:"repayType"`
	RepayTime        types.Time    `json:"repayTime"`
	OrderID          int64         `json:"orderId"`
}

// CryptoLoanRepayHistory stores crypto loan repayment history data
type CryptoLoanRepayHistory struct {
	Rows  []CryptoLoanRepayHistoryItem `json:"rows"`
	Total int64                        `json:"total"`
}

// CryptoLoanAdjustLTV stores crypto loan LTV adjustment data
type CryptoLoanAdjustLTV struct {
	LoanCoin       currency.Code `json:"loanCoin"`
	CollateralCoin currency.Code `json:"collateralCoin"`
	Direction      string        `json:"direction"`
	Amount         float64       `json:"amount,string"`
	CurrentLTV     float64       `json:"currentLTV,string"`
}

// CryptoLoanLTVAdjustmentItem stores crypto loan LTV adjustment item data
type CryptoLoanLTVAdjustmentItem struct {
	LoanCoin       currency.Code `json:"loanCoin"`
	CollateralCoin currency.Code `json:"collateralCoin"`
	Direction      string        `json:"direction"`
	Amount         float64       `json:"amount,string"`
	PreviousLTV    float64       `json:"preLTV,string"`
	AfterLTV       float64       `json:"afterLTV,string"`
	AdjustTime     types.Time    `json:"adjustTime"`
	OrderID        int64         `json:"orderId"`
}

// CryptoLoanLTVAdjustmentHistory stores crypto loan LTV adjustment history data
type CryptoLoanLTVAdjustmentHistory struct {
	Rows  []CryptoLoanLTVAdjustmentItem `json:"rows"`
	Total int64                         `json:"total"`
}

// LoanableAssetItem stores loanable asset item data
type LoanableAssetItem struct {
	LoanCoin                             currency.Code `json:"loanCoin"`
	SevenDayHourlyInterestRate           float64       `json:"_7dHourlyInterestRate,string"`
	SevenDayDailyInterestRate            float64       `json:"_7dDailyInterestRate,string"`
	FourteenDayHourlyInterest            float64       `json:"_14dHourlyInterestRate,string"`
	FourteenDayDailyInterest             float64       `json:"_14dDailyInterestRate,string"`
	ThirtyDayHourlyInterest              float64       `json:"_30dHourlyInterestRate,string"`
	ThirtyDayDailyInterest               float64       `json:"_30dDailyInterestRate,string"`
	NinetyDayHourlyInterest              float64       `json:"_90dHourlyInterestRate,string"`
	NinetyDayDailyInterest               float64       `json:"_90dDailyInterestRate,string"`
	OneHundredAndEightyDayHourlyInterest float64       `json:"_180dHourlyInterestRate,string"`
	OneHundredAndEightyDayDailyInterest  float64       `json:"_180dDailyInterestRate,string"`
	MinimumLimit                         float64       `json:"minLimit,string"`
	MaximumLimit                         float64       `json:"maxLimit,string"`
	VIPLevel                             int64         `json:"vipLevel"`
}

// LoanableAssetsData stores loanable assets data
type LoanableAssetsData struct {
	Rows  []LoanableAssetItem `json:"rows"`
	Total int64               `json:"total"`
}

// CollateralAssetItem stores collateral asset item data
type CollateralAssetItem struct {
	CollateralCoin currency.Code `json:"collateralCoin"`
	InitialLTV     float64       `json:"initialLTV,string"`
	MarginCallLTV  float64       `json:"marginCallLTV,string"`
	LiquidationLTV float64       `json:"liquidationLTV,string"`
	MaxLimit       float64       `json:"maxLimit,string"`
	VIPLevel       int64         `json:"vipLevel"`
}

// CollateralAssetData stores collateral asset data
type CollateralAssetData struct {
	Rows  []CollateralAssetItem `json:"rows"`
	Total int64                 `json:"total"`
}

// CollateralRepayRate stores collateral repayment rate data
type CollateralRepayRate struct {
	LoanCoin       currency.Code `json:"loanCoin"`
	CollateralCoin currency.Code `json:"collateralCoin"`
	RepayAmount    float64       `json:"repayAmount,string"`
	Rate           float64       `json:"rate,string"`
}

// CustomiseMarginCallItem stores customise margin call item data
type CustomiseMarginCallItem struct {
	OrderID         int64         `json:"orderId"`
	CollateralCoin  currency.Code `json:"collateralCoin"`
	PreMarginCall   float64       `json:"preMarginCall,string"`
	AfterMarginCall float64       `json:"afterMarginCall,string"`
	CustomiseTime   types.Time    `json:"customizeTime"`
}

// CustomiseMarginCall stores customise margin call data
type CustomiseMarginCall struct {
	Rows  []CustomiseMarginCallItem `json:"rows"`
	Total int64                     `json:"total"`
}

// FlexibleLoanBorrow stores a flexible loan borrow
type FlexibleLoanBorrow struct {
	LoanCoin         currency.Code `json:"loanCoin"`
	LoanAmount       float64       `json:"loanAmount,string"`
	CollateralCoin   currency.Code `json:"collateralCoin"`
	CollateralAmount float64       `json:"collateralAmount,string"`
	Status           string        `json:"status"`
}

// FlexibleLoanOngoingOrderItem stores a flexible loan ongoing order item
type FlexibleLoanOngoingOrderItem struct {
	LoanCoin         currency.Code `json:"loanCoin"`
	TotalDebt        float64       `json:"totalDebt,string"`
	CollateralCoin   currency.Code `json:"collateralCoin"`
	CollateralAmount float64       `json:"collateralAmount,string"`
	CurrentLTV       float64       `json:"currentLTV,string"`
}

// FlexibleLoanOngoingOrder stores flexible loan ongoing orders
type FlexibleLoanOngoingOrder struct {
	Rows  []FlexibleLoanOngoingOrderItem `json:"rows"`
	Total int64                          `json:"total"`
}

// FlexibleLoanBorrowHistoryItem stores a flexible loan borrow history item
type FlexibleLoanBorrowHistoryItem struct {
	LoanCoin                currency.Code `json:"loanCoin"`
	InitialLoanAmount       float64       `json:"initialLoanAmount,string"`
	CollateralCoin          currency.Code `json:"collateralCoin"`
	InitialCollateralAmount float64       `json:"initialCollateralAmount,string"`
	BorrowTime              types.Time    `json:"borrowTime"`
	Status                  string        `json:"status"`
}

// FlexibleLoanBorrowHistory stores flexible loan borrow history
type FlexibleLoanBorrowHistory struct {
	Rows  []FlexibleLoanBorrowHistoryItem `json:"rows"`
	Total int64                           `json:"total"`
}

// FlexibleLoanRepay stores a flexible loan repayment
type FlexibleLoanRepay struct {
	LoanCoin            currency.Code `json:"loanCoin"`
	CollateralCoin      currency.Code `json:"collateralCoin"`
	RemainingDebt       float64       `json:"remainingDebt,string"`
	RemainingCollateral float64       `json:"remainingCollateral,string"`
	FullRepayment       bool          `json:"fullRepayment"`
	CurrentLTV          float64       `json:"currentLTV,string"`
	RepayStatus         string        `json:"repayStatus"`
}

// FlexibleLoanRepayHistoryItem stores a flexible loan repayment history item
type FlexibleLoanRepayHistoryItem struct {
	LoanCoin         currency.Code `json:"loanCoin"`
	RepayAmount      float64       `json:"repayAmount,string"`
	CollateralCoin   currency.Code `json:"collateralCoin"`
	CollateralReturn float64       `json:"collateralReturn,string"`
	RepayStatus      string        `json:"repayStatus"`
	RepayTime        types.Time    `json:"repayTime"`
}

// FlexibleLoanRepayHistory stores flexible loan repayment history
type FlexibleLoanRepayHistory struct {
	Rows  []FlexibleLoanRepayHistoryItem `json:"rows"`
	Total int64                          `json:"total"`
}

// FlexibleLoanAdjustLTV stores a flexible loan LTV adjustment
type FlexibleLoanAdjustLTV struct {
	LoanCoin       currency.Code `json:"loanCoin"`
	CollateralCoin currency.Code `json:"collateralCoin"`
	Direction      string        `json:"direction"`
	Amount         float64       `json:"amount,string"` // docs error: API actually returns "amount" instead of "adjustedAmount"
	CurrentLTV     float64       `json:"currentLTV,string"`
	Status         string        `json:"status"`
}

// FlexibleLoanLTVAdjustmentHistoryItem stores a flexible loan LTV adjustment history item
type FlexibleLoanLTVAdjustmentHistoryItem struct {
	LoanCoin         currency.Code `json:"loanCoin"`
	CollateralCoin   currency.Code `json:"collateralCoin"`
	Direction        string        `json:"direction"`
	CollateralAmount float64       `json:"collateralAmount,string"`
	PreviousLTV      float64       `json:"preLTV,string"`
	AfterLTV         float64       `json:"afterLTV,string"`
	AdjustTime       types.Time    `json:"adjustTime"`
}

// FlexibleLoanLTVAdjustmentHistory stores flexible loan LTV adjustment history
type FlexibleLoanLTVAdjustmentHistory struct {
	Rows  []FlexibleLoanLTVAdjustmentHistoryItem `json:"rows"`
	Total int64                                  `json:"total"`
}

// FlexibleLoanAssetsDataItem stores a flexible loan asset data item
type FlexibleLoanAssetsDataItem struct {
	LoanCoin             currency.Code `json:"loanCoin"`
	FlexibleInterestRate float64       `json:"flexibleInterestRate,string"`
	FlexibleMinLimit     float64       `json:"flexibleMinLimit,string"`
	FlexibleMaxLimit     float64       `json:"flexibleMaxLimit,string"`
}

// FlexibleLoanAssetsData stores flexible loan asset data
type FlexibleLoanAssetsData struct {
	Rows  []FlexibleLoanAssetsDataItem `json:"rows"`
	Total int64                        `json:"total"`
}

// FlexibleCollateralAssetsDataItem stores a flexible collateral asset data item
type FlexibleCollateralAssetsDataItem struct {
	CollateralCoin currency.Code `json:"collateralCoin"`
	InitialLTV     float64       `json:"initialLTV,string"`
	MarginCallLTV  float64       `json:"marginCallLTV,string"`
	LiquidationLTV float64       `json:"liquidationLTV,string"`
	MaxLimit       float64       `json:"maxLimit,string"`
}

// FlexibleCollateralAssetsData stores flexible collateral asset data
type FlexibleCollateralAssetsData struct {
	Rows  []FlexibleCollateralAssetsDataItem `json:"rows"`
	Total int64                              `json:"total"`
}
