package binance

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
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

	// Futures channels
	contractInfoAllChan = "!contractInfo"
	forceOrderAllChan   = "!forceOrder@arr"
	bookTickerAllChan   = "!bookTicker"
	tickerAllChan       = "!ticker@arr"
	miniTickerAllChan   = "!miniTicker@arr"
	aggTradeChan        = "@aggTrade"
	depthChan           = "@depth"
	markPriceChan       = "@markPrice"
	tickerChan          = "@ticker"
	klineChan           = "@kline"
	miniTickerChan      = "@miniTicker"
	forceOrderChan      = "@forceOrder"
	continuousKline     = "continuousKline"

	// USDT Marigined futures
	markPriceAllChan   = "!markPrice@arr"
	assetIndexChan     = "@assetIndex"
	bookTickersChan    = "@bookTickers"
	assetIndexAllChan  = "!assetIndex@arr"
	compositeIndexChan = "@compositeIndex"

	// Coin Margined futures
	indexPriceCFuturesChan      = "@indexPrice"
	bookTickerCFuturesChan      = "@bookTicker"
	indexPriceKlineCFuturesChan = "@indexPriceKline"
	markPriceKlineCFuturesChan  = "@markPriceKline"
)

var (
	errLoanCoinMustBeSet                      = errors.New("loan coin must bet set")
	errLoanTermMustBeSet                      = errors.New("loan term must be set")
	errCollateralCoinMustBeSet                = errors.New("collateral coin must be set")
	errEitherLoanOrCollateralAmountsMustBeSet = errors.New("either loan or collateral amounts must be set")
	errNilArgument                            = errors.New("nil argument")
	errTimestampInfoRequired                  = errors.New("timestamp information is required")
	errListenKeyIsRequired                    = errors.New("listen key is required")
	errValidEmailRequired                     = errors.New("valid email address is required")
	errPageNumberRequired                     = errors.New("page number is required")
	errLimitNumberRequired                    = errors.New("invalid limit")
	errEmptySubAccountEPIKey                  = errors.New("invalid sub-account API key")
	errInvalidFuturesType                     = errors.New("invalid futures types")
	errInvalidAccountType                     = errors.New("invalid account type specified")
	errProductIDIsRequired                    = errors.New("product ID is required")
	errProjectIDRequired                      = errors.New("project ID is required")
	errPlanIDRequired                         = errors.New("plan ID  is required")
	errIndexIDIsRequired                      = errors.New("index ID is required")
	errTokenRequired                          = errors.New("token is required")
	errTransferAlgorithmRequired              = errors.New("transfer algorithm is required")
	errUsernameRequired                       = errors.New("user name is required")
	errTransferTypeRequired                   = errors.New("transfer type is required")
	errNameRequired                           = errors.New("name is required")
	errTradeTypeRequired                      = errors.New("trade type is required")
	errPositionIDRequired                     = errors.New("position ID is required")
	errOptionTypeRequired                     = errors.New("optionType is required")
	errInvalidPositionSide                    = errors.New("invalid positionSide")
	errInvalidWorkingType                     = errors.New("invalid workingType")
	errPageSizeRequired                       = errors.New("page size is required")
	errInvalidSubscriptionStartTime           = errors.New("invalid subscription start time")
	errPortfolioDetailRequired                = errors.New("portfolio detail is required")
	errPlanTypeRequired                       = errors.New("planType is required")
	errTransactionIDRequired                  = errors.New("transaction ID is required")
	errRequestIDRequired                      = errors.New("request ID is required")
	errStartTimeRequired                      = errors.New("start time is required")
	errStrategyTypeRequired                   = errors.New("strategy type is required")
	errReferenceNumberRequired                = errors.New("reference number is required")
	errExpiedTypeRequired                     = errors.New("expiredType is required")
	errQuoteIDRequired                        = errors.New("quote ID is required")
	errAccountIDRequired                      = errors.New("account ID is required")
	errAccountRequired                        = errors.New("account information required")
	errConfigIDRequired                       = errors.New("config ID is required")
	errLendingTypeRequired                    = errors.New("lending type is required")
	errEmptyCurrencyCodes                     = errors.New("assetNames are required")
	errSourceTypeRequired                     = errors.New("source type required")
	errInvalidPercentageAmount                = errors.New("invalid percentage amount")
	errHashRateRequired                       = errors.New("hash rate is required")
	errCodeRequired                           = errors.New("code is required")
	errAddressRequired                        = errors.New("address is required")
	errInvalidSubscriptionCycle               = errors.New("invalid subscription cycle")
	errDurationRequired                       = errors.New("duration is required")
	errCostRequired                           = errors.New("cost must be greater than 0")
	errInvalidTransactionType                 = errors.New("invalid transaction type")
	errPossibleValuesRequired                 = errors.New("urgency field is required")
	errPlanStatusRequired                     = errors.New("plan status is required")
	errUsageTypeRequired                      = errors.New("usage type is required")
	errMarginCallValueRequired                = errors.New("margin call value required")
	errTimeInForceRequired                    = errors.New("time in force required")
	errIncomeTypeRequired                     = errors.New("invalid incomeType")
	errInvalidNewOrderResponseType            = errors.New("invalid new order response type")
	errInvalidAutoCloseType                   = errors.New("invalid auto close type")
	errDownloadIDRequired                     = errors.New("downloadId is required")
	errMarginChangeTypeInvalid                = errors.New("invalid margin changeType")
)

var subscriptionCycleList = []string{"H1", "H4", "H8", "H12", "WEEKLY", "DAILY", "MONTHLY", "BI_WEEKLY"}

// TransferTypes represents asset transfer types
type TransferTypes uint8

const (
	ttMainUMFuture TransferTypes = iota + 1
	ttMainCMFuture
	ttMainMargin
	ttUMFutureMain
	ttUMFutureMargin
	ttCMFutureMain
	ttCMFutureMargin
	ttMarginMain
	ttMarginUMFuture
	ttMarginCMFuture
	ttIsolatedMarginMargin
	ttMarginIsolatedMargin
	ttIsolatedMarginIsolatedMargin
	ttMainFunding
	ttFundingMain
	ttFundingUMFuture
	ttUMFutureFunding
	ttMarginFunding
	ttFundingMargin
	ttFundingCMFuture
	ttCMFutureFunding
	ttMainOption
	ttOptionMain
	ttUMFutureOption
	ttOptionUMFuture
	ttMarginOption
	ttOptionMargin
	ttFundingOption
	ttOptionFunding
	ttMainPortfolioMargin
	ttPortfolioMarginMain
	ttMainIsolatedMargin
	ttIsolatedMarginMain
)

// String returns a string representation of transfer type
func (a TransferTypes) String() string {
	switch a {
	case ttMainUMFuture:
		// Spot account transfer to USDⓈ-M Futures account
		return "MAIN_UMFUTURE"
	case ttMainCMFuture:
		// Spot account transfer to COIN-M Futures account
		return "MAIN_CMFUTURE"
	case ttMainMargin:
		// Spot account transfer to Margin（cross）account
		return "MAIN_MARGIN"
	case ttUMFutureMain:
		// USDⓈ-M Futures account transfer to Spot account
		return "UMFUTURE_MAIN"
	case ttUMFutureMargin:
		// USDⓈ-M Futures account transfer to Margin（cross）account
		return "UMFUTURE_MARGIN"
	case ttCMFutureMain:
		// COIN-M Futures account transfer to Spot account
		return "CMFUTURE_MAIN"
	case ttCMFutureMargin:
		// COIN-M Futures account transfer to Margin(cross) account
		return "CMFUTURE_MARGIN"
	case ttMarginMain:
		// Margin（cross）account transfer to Spot account
		return "MARGIN_MAIN"
	case ttMarginUMFuture:
		// Margin（cross）account transfer to USDⓈ-M Futures
		return "MARGIN_UMFUTURE"
	case ttMarginCMFuture:
		// Margin（cross）account transfer to COIN-M Futures
		return "MARGIN_CMFUTURE"
	case ttIsolatedMarginMargin:
		// Isolated margin account transfer to Margin(cross) account
		return "ISOLATEDMARGIN_MARGIN"
	case ttMarginIsolatedMargin:
		// Margin(cross) account transfer to Isolated margin account
		return "MARGIN_ISOLATEDMARGIN"
	case ttIsolatedMarginIsolatedMargin:
		// Isolated margin account transfer to Isolated margin account
		return "ISOLATEDMARGIN_ISOLATEDMARGIN"
	case ttMainFunding:
		// Spot account transfer to Funding account
		return "MAIN_FUNDING"
	case ttFundingMain:
		// Funding account transfer to Spot account
		return "FUNDING_MAIN"
	case ttFundingUMFuture:
		// Funding account transfer to UMFUTURE account
		return "FUNDING_UMFUTURE"
	case ttUMFutureFunding:
		// UMFUTURE account transfer to Funding account
		return "UMFUTURE_FUNDING"
	case ttMarginFunding:
		// MARGIN account transfer to Funding account
		return "MARGIN_FUNDING"
	case ttFundingMargin:
		// Funding account transfer to Margin account
		return "FUNDING_MARGIN"
	case ttFundingCMFuture:
		// Funding account transfer to CMFUTURE account
		return "FUNDING_CMFUTURE"
	case ttCMFutureFunding:
		// CMFUTURE account transfer to Funding account
		return "CMFUTURE_FUNDING"
	case ttMainOption:
		// Spot account transfer to Options account
		return "MAIN_OPTION"
	case ttOptionMain:
		// Options account transfer to Spot account
		return "OPTION_MAIN"
	case ttUMFutureOption:
		// USDⓈ-M Futures account transfer to Options account
		return "UMFUTURE_OPTION"
	case ttOptionUMFuture:
		// Options account transfer to USDⓈ-M Futures account
		return "OPTION_UMFUTURE"
	case ttMarginOption:
		// Margin（cross）account transfer to Options account
		return "MARGIN_OPTION"
	case ttOptionMargin:
		// Options account transfer to Margin（cross）account
		return "OPTION_MARGIN"
	case ttFundingOption:
		// Funding account transfer to Options account
		return "FUNDING_OPTION"
	case ttOptionFunding:
		// Options account transfer to Funding account
		return "OPTION_FUNDING"
	case ttMainPortfolioMargin:
		// Spot account transfer to Portfolio Margin account
		return "MAIN_PORTFOLIO_MARGIN"
	case ttPortfolioMarginMain:
		// Portfolio Margin account transfer to Spot account
		return "PORTFOLIO_MARGIN_MAIN"
	case ttMainIsolatedMargin:
		// Spot account transfer to Isolated margin account
		return "MAIN_ISOLATED_MARGIN"
	case ttIsolatedMarginMain:
		// Isolated margin account transfer to Spot account
		return "ISOLATED_MARGIN_MAIN"
	default:
		return ""
	}
}

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
	Code            int              `json:"code"`
	Msg             string           `json:"msg"`
	Timezone        string           `json:"timezone"`
	ServerTime      types.Time       `json:"serverTime"`
	RateLimits      []*RateLimitItem `json:"rateLimits"`
	ExchangeFilters interface{}      `json:"exchangeFilters"`
	Symbols         []*struct {
		Symbol                          string        `json:"symbol"`
		Status                          string        `json:"status"`
		BaseAsset                       string        `json:"baseAsset"`
		BaseAssetPrecision              int64         `json:"baseAssetPrecision"`
		QuoteAsset                      string        `json:"quoteAsset"`
		QuotePrecision                  int64         `json:"quotePrecision"`
		OrderTypes                      []string      `json:"orderTypes"`
		IcebergAllowed                  bool          `json:"icebergAllowed"`
		OCOAllowed                      bool          `json:"ocoAllowed"`
		QuoteOrderQtyMarketAllowed      bool          `json:"quoteOrderQtyMarketAllowed"`
		IsSpotTradingAllowed            bool          `json:"isSpotTradingAllowed"`
		IsMarginTradingAllowed          bool          `json:"isMarginTradingAllowed"`
		Filters                         []*filterData `json:"filters"`
		Permissions                     []string      `json:"permissions"`
		QuoteAssetPrecision             int64         `json:"quoteAssetPrecision"`
		AllowTrailingStop               bool          `json:"allowTrailingStop"`
		CancelReplaceAllowed            bool          `json:"cancelReplaceAllowed"`
		PermissionSets                  [][]string    `json:"permissionSets"`
		DefaultSelfTradePreventionMode  string        `json:"defaultSelfTradePreventionMode"`
		AllowedSelfTradePreventionModes []string      `json:"allowedSelfTradePreventionModes"`
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

// OrderBookDataRequestParams represents Klines request data.
type OrderBookDataRequestParams struct {
	Symbol currency.Pair `json:"symbol"` // Required field; example LTCBTC,BTCUSDT
	Limit  int64         `json:"limit"`  // Default 100; max 1000. Valid limits:[5, 10, 20, 50, 100, 500, 1000]
}

// OrderbookItem stores an individual orderbook item
type OrderbookItem struct {
	Price    float64
	Quantity float64
}

// OrderBookData is resp data from orderbook endpoint
type OrderBookData struct {
	Code         int64             `json:"code"`
	Msg          string            `json:"msg"`
	LastUpdateID int64             `json:"lastUpdateId"`
	Bids         [][2]types.Number `json:"bids"`
	Asks         [][2]types.Number `json:"asks"`
}

// OrderBook actual structured data that can be used for orderbook
type OrderBook struct {
	Symbol       string
	LastUpdateID int64
	Code         int64
	Msg          string
	Bids         []OrderbookItem
	Asks         []OrderbookItem
}

// DepthUpdateParams is used as an embedded type for WebsocketDepthStream
type DepthUpdateParams []struct {
	PriceLevel float64
	Quantity   float64
	ignore     []interface{}
}

// WebsocketDepthStream is the difference for the update depth stream
type WebsocketDepthStream struct {
	Event         string            `json:"e"`
	Timestamp     types.Time        `json:"E"`
	Pair          string            `json:"s"`
	FirstUpdateID int64             `json:"U"`
	LastUpdateID  int64             `json:"u"`
	UpdateBids    [][2]types.Number `json:"b"`
	UpdateAsks    [][2]types.Number `json:"a"`
}

// RecentTradeRequestParams represents Klines request data.
type RecentTradeRequestParams struct {
	Symbol currency.Pair `json:"symbol"` // Required field. example LTCBTC, BTCUSDT
	Limit  int64         `json:"limit"`  // Default 500; max 500.
	FromID int64         `json:"fromId,omitempty"`
}

// RecentTrade holds recent trade data
type RecentTrade struct {
	ID           int64      `json:"id"`
	Price        float64    `json:"price,string"`
	Quantity     float64    `json:"qty,string"`
	QuoteQty     string     `json:"quoteQty"`
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
	Symbol string // Required field; example LTCBTC, BTCUSDT
	// The first trade to retrieve
	FromID int64
	// The API seems to accept (start and end time) or FromID and no other combinations
	StartTime time.Time
	EndTime   time.Time
	// Default 500; max 1000.
	Limit int
}

// WsAggregateTradeRequestParams holds request parameters for aggregate trades
type WsAggregateTradeRequestParams struct {
	Symbol    string `json:"symbol"`
	FromID    int64  `json:"fromId,omitempty"`
	Limit     int64  `json:"limit,omitempty"`
	StartTime int64  `json:"startTime,omitempty"`
	EndTime   int64  `json:"endTime,omitempty"`
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

// UFuturesAggregatedTrade represents usdt futures aggregated trade information
type UFuturesAggregatedTrade struct {
	EventType        string       `json:"e"`
	EventTime        types.Time   `json:"E"`
	Symbol           string       `json:"s"`
	AggregateTradeID int64        `json:"a"`
	Price            types.Number `json:"p"`
	Quantity         types.Number `json:"q"`
	FirstTradeID     int64        `json:"f"`
	LastTradeID      int64        `json:"l"`
	TradeTime        types.Time   `json:"T"`
	MarketMaker      bool         `json:"m"`
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

// PriceChangesWrapper to be used when the response is either a single PriceChangeStats instance or a slice.
type PriceChangesWrapper []PriceChangeStats

// PriceChangeStats contains statistics for the last 24 hours trade
type PriceChangeStats struct {
	Symbol             string       `json:"symbol"`
	PriceChange        types.Number `json:"priceChange"`
	PriceChangePercent types.Number `json:"priceChangePercent"`
	WeightedAvgPrice   types.Number `json:"weightedAvgPrice"`
	PrevClosePrice     types.Number `json:"prevClosePrice"`
	LastPrice          types.Number `json:"lastPrice"`
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

	LastQty  types.Number `json:"lastQty"`
	BidPrice types.Number `json:"bidPrice"`
	BidQty   types.Number `json:"bidQty"`
	AskPrice types.Number `json:"askPrice"`
	AskQty   types.Number `json:"askQty"`
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
	Code            int64      `json:"code"`
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
		Quantity        float64 `json:"qty,string"`
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

// TradeOrder holds query order data
// Note that some fields are optional and included only for orders that set them.
type TradeOrder struct {
	Code                    int64        `json:"code"`
	Msg                     string       `json:"msg"`
	Symbol                  string       `json:"symbol"`
	OrderID                 int64        `json:"orderId"`
	OrderListID             int64        `json:"orderListId"`
	ClientOrderID           string       `json:"clientOrderId"`
	Price                   types.Number `json:"price"`
	OrigQty                 types.Number `json:"origQty"`
	ExecutedQty             types.Number `json:"executedQty"`
	CummulativeQuoteQty     types.Number `json:"cummulativeQuoteQty"`
	Status                  string       `json:"status"`
	TimeInForce             string       `json:"timeInForce"`
	Type                    string       `json:"type"`
	Side                    string       `json:"side"`
	IsWorking               bool         `json:"isWorking"`
	StopPrice               types.Number `json:"stopPrice"`
	IcebergQty              types.Number `json:"icebergQty"`
	Time                    types.Time   `json:"time"`
	UpdateTime              types.Time   `json:"updateTime"`
	WorkingTime             types.Time   `json:"workingTime"`
	OrigQuoteOrderQty       types.Number `json:"origQuoteOrderQty"`
	SelfTradePreventionMode string       `json:"selfTradePreventionMode"`
	OrigClientOrderID       string       `json:"origClientOrderId"`
	TransactTime            types.Time   `json:"transactTime"`

	PreventedMatchID  int64        `json:"preventedMatchId"`
	PreventedQuantity types.Number `json:"preventedQuantity"`

	IsIsolated bool `json:"isIsolated"`
}

// SymbolOrders represents a symbol and orders related to the symbol
type SymbolOrders struct {
	Symbol                  string       `json:"symbol"`
	OrigClientOrderID       string       `json:"origClientOrderId,omitempty"`
	OrderID                 int64        `json:"orderId,omitempty"`
	OrderListID             int64        `json:"orderListId"`
	ClientOrderID           string       `json:"clientOrderId,omitempty"`
	TransactTime            types.Time   `json:"transactTime,omitempty"`
	Price                   types.Number `json:"price,omitempty"`
	OrigQty                 types.Number `json:"origQty,omitempty"`
	ExecutedQty             types.Number `json:"executedQty,omitempty"`
	CummulativeQuoteQty     types.Number `json:"cummulativeQuoteQty,omitempty"`
	Status                  string       `json:"status,omitempty"`
	TimeInForce             string       `json:"timeInForce,omitempty"`
	Type                    string       `json:"type,omitempty"`
	Side                    string       `json:"side,omitempty"`
	SelfTradePreventionMode string       `json:"selfTradePreventionMode,omitempty"`
	ContingencyType         string       `json:"contingencyType,omitempty"`
	ListStatusType          string       `json:"listStatusType,omitempty"`
	ListOrderStatus         string       `json:"listOrderStatus,omitempty"`
	ListClientOrderID       string       `json:"listClientOrderId,omitempty"`
	TransactionTime         int64        `json:"transactionTime,omitempty"`
	Orders                  []struct {
		Symbol        string `json:"symbol"`
		OrderID       int64  `json:"orderId"`
		ClientOrderID string `json:"clientOrderId"`
	} `json:"orders,omitempty"`
	OrderReports []struct {
		Symbol                  string       `json:"symbol"`
		OrigClientOrderID       string       `json:"origClientOrderId"`
		OrderID                 int          `json:"orderId"`
		OrderListID             int          `json:"orderListId"`
		ClientOrderID           string       `json:"clientOrderId"`
		TransactTime            int64        `json:"transactTime"`
		Price                   types.Number `json:"price"`
		OrigQty                 types.Number `json:"origQty"`
		ExecutedQty             types.Number `json:"executedQty"`
		CummulativeQuoteQty     types.Number `json:"cummulativeQuoteQty"`
		Status                  string       `json:"status"`
		TimeInForce             string       `json:"timeInForce"`
		Type                    string       `json:"type"`
		Side                    string       `json:"side"`
		StopPrice               types.Number `json:"stopPrice,omitempty"`
		IcebergQty              types.Number `json:"icebergQty"`
		SelfTradePreventionMode string       `json:"selfTradePreventionMode"`
	} `json:"orderReports,omitempty"`
}

// CancelReplaceOrderParams represents a request parameter to cancel an existing order and send a new order.
type CancelReplaceOrderParams struct {
	Symbol    string `json:"symbol"`
	Side      string `json:"side"`
	OrderType string `json:"type"`

	// The allowed values are:
	// STOP_ON_FAILURE - If the cancel request fails, the new order placement will not be attempted.
	// ALLOW_FAILURE - new order placement will be attempted even if cancel request fails.
	CancelReplaceMode       string  `json:"cancelReplaceMode"`
	TimeInForce             string  `json:"timeInForce,omitempty"`
	Quantity                float64 `json:"quantity,omitempty"`
	QuoteOrderQuantity      float64 `json:"quoteOrderQty,omitempty"`
	Price                   float64 `json:"price,omitempty"`
	CancelNewClientOrderID  string  `json:"cancelNewClientOrderId,omitempty"`
	CancelOrigClientOrderID string  `json:"cancelOrigClientOrderId,omitempty"`
	CancelOrderID           string  `json:"cancelOrderId,omitempty"`
	NewClientOrderID        string  `json:"newClientOrderId,omitempty"`
	StrategyID              int64   `json:"strategyId,omitempty"`
	StrategyType            int64   `json:"strategyType,omitempty"`
	StopPrice               float64 `json:"stopPrice,omitempty"`
	TrailingDelta           int64   `json:"trailingDelta,omitempty"`
	IcebergQuantity         float64 `json:"icebergQty,omitempty"`

	// NewOrderRespType Allowed values: ACK, RESULT, FULL
	// MARKET and LIMIT orders types default to FULL; all other orders default to ACK
	NewOrderRespType        string `json:"newOrderRespType,omitempty"`
	SelfTradePreventionMode string `json:"selfTradePreventionMode,omitempty"`
	CancelRestrictions      string `json:"cancelRestrictions,omitempty"`
}

// CancelAndReplaceResponse represents an order cancellation and replacement response.
type CancelAndReplaceResponse struct {
	CancelResult   string `json:"cancelResult"`
	NewOrderResult string `json:"newOrderResult"`
	CancelResponse struct {
		Symbol                  string       `json:"symbol"`
		OrigClientOrderID       string       `json:"origClientOrderId"`
		OrderID                 int64        `json:"orderId"`
		OrderListID             int64        `json:"orderListId"`
		ClientOrderID           string       `json:"clientOrderId"`
		TransactTime            int64        `json:"transactTime"`
		Price                   types.Number `json:"price"`
		OrigQty                 types.Number `json:"origQty"`
		ExecutedQty             types.Number `json:"executedQty"`
		CummulativeQuoteQty     types.Number `json:"cummulativeQuoteQty"`
		Status                  string       `json:"status"`
		TimeInForce             string       `json:"timeInForce"`
		Type                    string       `json:"type"`
		Side                    string       `json:"side"`
		SelfTradePreventionMode string       `json:"selfTradePreventionMode"`
	} `json:"cancelResponse"`
	NewOrderResponse struct {
		Symbol                  string `json:"symbol"`
		OrderID                 int    `json:"orderId"`
		OrderListID             int    `json:"orderListId"`
		ClientOrderID           string `json:"clientOrderId"`
		TransactTime            int64  `json:"transactTime"`
		Price                   string `json:"price"`
		OrigQty                 string `json:"origQty"`
		ExecutedQty             string `json:"executedQty"`
		CummulativeQuoteQty     string `json:"cummulativeQuoteQty"`
		Status                  string `json:"status"`
		TimeInForce             string `json:"timeInForce"`
		Type                    string `json:"type"`
		Side                    string `json:"side"`
		WorkingTime             int64  `json:"workingTime"`
		Fills                   []any  `json:"fills"`
		SelfTradePreventionMode string `json:"selfTradePreventionMode"`
	} `json:"newOrderResponse"`
}

// Balance holds query order data
type Balance struct {
	Asset  string          `json:"asset"`
	Free   decimal.Decimal `json:"free"`
	Locked decimal.Decimal `json:"locked"`
}

// Account holds the account data
type Account struct {
	UID              int64        `json:"uid"`
	MakerCommission  types.Number `json:"makerCommission"`
	TakerCommission  types.Number `json:"takerCommission"`
	BuyerCommission  types.Number `json:"buyerCommission"`
	SellerCommission types.Number `json:"sellerCommission"`
	CanTrade         bool         `json:"canTrade"`
	CanWithdraw      bool         `json:"canWithdraw"`
	CanDeposit       bool         `json:"canDeposit"`
	CommissionRates  struct {
		Maker  types.Number `json:"maker"`
		Taker  types.Number `json:"taker"`
		Buyer  types.Number `json:"buyer"`
		Seller types.Number `json:"seller"`
	} `json:"commissionRates"`
	Brokered                   bool       `json:"brokered"`
	RequireSelfTradePrevention bool       `json:"requireSelfTradePrevention"`
	PreventSor                 bool       `json:"preventSor"`
	UpdateTime                 types.Time `json:"updateTime"`
	AccountType                string     `json:"accountType"`
	Balances                   []Balance  `json:"balances"`
	Permissions                []string   `json:"permissions"`
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
	Symbol         currency.Pair `json:"symbol"`             // Required field; example LTCBTC, BTCUSDT
	Interval       string        `json:"interval,omitempty"` // Time interval period
	Limit          int64         `json:"limit,omitempty"`    // Default 500; max 500.
	StartTime      time.Time     `json:"-"`
	EndTime        time.Time     `json:"-"`
	Timezone       string        `json:"timeZone,omitempty"`
	StartTimestamp int64         `json:"startTime,omitempty"`
	EndTimestamp   int64         `json:"endTime,omitempty"`
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

// AssetsDust represents assets that can be converted into BNB
type AssetsDust struct {
	Details []struct {
		Asset            string `json:"asset"`
		AssetFullName    string `json:"assetFullName"`
		AmountFree       string `json:"amountFree"`
		ToBTC            string `json:"toBTC"`
		ToBNB            string `json:"toBNB"`
		ToBNBOffExchange string `json:"toBNBOffExchange"`
		Exchange         string `json:"exchange"`
	} `json:"details"`
	TotalTransferBtc   string `json:"totalTransferBtc"`
	TotalTransferBNB   string `json:"totalTransferBNB"`
	DribbletPercentage string `json:"dribbletPercentage"`
}

// Dusts represents a response after converting assets to BNB
type Dusts struct {
	TotalServiceCharge string `json:"totalServiceCharge"`
	TotalTransfered    string `json:"totalTransfered"`
	TransferResult     []struct {
		Amount              types.Number `json:"amount"`
		FromAsset           string       `json:"fromAsset"`
		OperateTime         types.Time   `json:"operateTime"`
		ServiceChargeAmount types.Number `json:"serviceChargeAmount"`
		TranID              int64        `json:"tranId"`
		TransferedAmount    types.Number `json:"transferedAmount"`
	} `json:"transferResult"`
}

// AssetDividendRecord represents an asset dividend record.
type AssetDividendRecord struct {
	Rows []struct {
		ID      int64        `json:"id"`
		Amount  types.Number `json:"amount"`
		Asset   string       `json:"asset"`
		DivTime types.Time   `json:"divTime"`
		EnInfo  string       `json:"enInfo"`
		TranID  int64        `json:"tranId"`
	} `json:"rows"`
	Total int64 `json:"total"`
}

// DividendAsset represents details of assets
type DividendAsset struct {
	MinWithdrawAmount types.Number `json:"minWithdrawAmount"`
	DepositStatus     bool         `json:"depositStatus"`
	WithdrawFee       types.Number `json:"withdrawFee"`
	WithdrawStatus    bool         `json:"withdrawStatus"`
}

// TradeFee represents a trading fee for an asset.
type TradeFee struct {
	Symbol          string       `json:"symbol"`
	MakerCommission types.Number `json:"makerCommission"`
	TakerCommission types.Number `json:"takerCommission"`
}

// UniversalTransferHistory query user universal transfer history
type UniversalTransferHistory struct {
	Total int64          `json:"total"`
	Rows  []TransferItem `json:"rows"`
}

// TransferItem represents a universal transfer information
type TransferItem struct {
	Asset     string       `json:"asset"`
	Amount    types.Number `json:"amount"`
	Type      string       `json:"type"`
	Status    string       `json:"status"`
	TranID    int64        `json:"tranId"`
	Timestamp types.Time   `json:"timestamp"`
}

// FundingAsset represents a funding asset
type FundingAsset struct {
	Asset        string       `json:"asset"`
	Free         types.Number `json:"free"`
	Locked       types.Number `json:"locked"`
	Freeze       types.Number `json:"freeze"`
	Withdrawing  string       `json:"withdrawing"`
	IPoable      string       `json:"ipoable"`
	BtcValuation string       `json:"btcValuation"`
}

// AssetConverResponse represents a response after converting a BUSD
type AssetConverResponse struct {
	TransactionID string `json:"tranId"`
	Status        string `json:"status"`
}

// BUSDConvertHistory represents a BUSD conversion history
type BUSDConvertHistory struct {
	Total int `json:"total"`
	Rows  []struct {
		TransactionID  int64        `json:"tranId"`
		Type           int64        `json:"type"`
		Time           types.Time   `json:"time"`
		DeductedAsset  string       `json:"deductedAsset"`
		DeductedAmount types.Number `json:"deductedAmount"`
		TargetAsset    string       `json:"targetAsset"`
		TargetAmount   types.Number `json:"targetAmount"`
		Status         string       `json:"status"`
		AccountType    string       `json:"accountType"`
	} `json:"rows"`
}

// CloudMiningPR cloud-mining payment and refund history
type CloudMiningPR struct {
	Total int64 `json:"total"`
	Rows  []struct {
		CreateTime types.Time   `json:"createTime"`
		TranID     int64        `json:"tranId"`
		Type       int64        `json:"type"`
		Asset      string       `json:"asset"`
		Amount     types.Number `json:"amount"`
		Status     string       `json:"status"`
	} `json:"rows"`
}

// APIKeyPermissions represents the API key permissions
type APIKeyPermissions struct {
	IPRestrict                   bool       `json:"ipRestrict"`
	CreateTime                   types.Time `json:"createTime"`
	EnableInternalTransfer       bool       `json:"enableInternalTransfer"`
	EnableFutures                bool       `json:"enableFutures"`
	EnablePortfolioMarginTrading bool       `json:"enablePortfolioMarginTrading"`
	EnableVanillaOptions         bool       `json:"enableVanillaOptions"`
	PermitsUniversalTransfer     bool       `json:"permitsUniversalTransfer"`
	EnableReading                bool       `json:"enableReading"`
	EnableSpotAndMarginTrading   bool       `json:"enableSpotAndMarginTrading"`
	EnableWithdrawals            bool       `json:"enableWithdrawals"`
	EnableMargin                 bool       `json:"enableMargin"`
}

// AutoConvertingStableCoins represents auto-conversion settings in deposit/withdrawal
type AutoConvertingStableCoins struct {
	ConvertEnabled bool              `json:"convertEnabled"`
	Coins          []string          `json:"coins"`
	ExchangeRates  map[string]string `json:"exchangeRates"`
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
	CanDeposit       bool       `json:"D"`
	CanTrade         bool       `json:"T"`
	CanWithdraw      bool       `json:"W"`
	EventTime        types.Time `json:"E"`
	LastUpdated      types.Time `json:"u"`
	BuyerCommission  float64    `json:"b"`
	MakerCommission  float64    `json:"m"`
	SellerCommission float64    `json:"s"`
	TakerCommission  float64    `json:"t"`
	EventType        string     `json:"e"`
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
	EventTime   types.Time `json:"E"`
	LastUpdated types.Time `json:"u"`
	EventType   string     `json:"e"`
}

type wsBalanceUpdate struct {
	Stream string              `json:"stream"`
	Data   WsBalanceUpdateData `json:"data"`
}

// WsBalanceUpdateData defines websocket account balance data
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
	WorkingTime                       types.Time `json:"W"`
	CumulativeQuoteTransactedQuantity float64    `json:"Z,string"`
	LastQuoteAssetTransactedQuantity  float64    `json:"Y,string"`
	QuoteOrderQuantity                float64    `json:"Q,string"`
}

type wsListStatus struct {
	Stream string           `json:"stream"`
	Data   WsListStatusData `json:"data"`
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

// UFuturesOrderbook holds orderbook data for usdt assets
type UFuturesOrderbook struct {
	Stream string `json:"stream"`
	Data   struct {
		EventType               string            `json:"e"`
		EventTime               types.Time        `json:"E"`
		TransactionTime         types.Time        `json:"T"`
		Symbol                  string            `json:"s"`
		FirstUpdateID           int64             `json:"U"`
		FinalUpdateID           int64             `json:"u"`
		FinalUpdateIDLastStream int64             `json:"pu"`
		Bids                    [][2]types.Number `json:"b"`
		Asks                    [][2]types.Number `json:"a"`
	} `json:"data"`
}

// UFuturesKline updates to the current klines/candlestick
type UFuturesKline struct {
	EventType string     `json:"e"`
	EventTime types.Time `json:"E"`
	Symbol    string     `json:"s"`
	KlineData struct {
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
		BaseVolume               types.Number `json:"v"`
		NumberOfTrades           int64        `json:"n"`
		IsKlineClosed            bool         `json:"x"`
		QuoteVolume              types.Number `json:"q"`
		TakerBuyBaseAssetVolume  types.Number `json:"V"`
		TakerBuyQuoteAssetVolume types.Number `json:"Q"`
		B                        string       `json:"B"`
	} `json:"k"`
}

// FuturesMarkPrice represents usdt futures mark price and funding rate for a single symbol pushed every 3
type FuturesMarkPrice struct {
	EventType            string       `json:"e"`
	EventTime            types.Time   `json:"E"`
	Symbol               string       `json:"s"`
	MarkPrice            types.Number `json:"p"`
	IndexPrice           types.Number `json:"i"`
	EstimatedSettlePrice types.Number `json:"P"` // Estimated Settle Price, only useful in the last hour before the settlement starts
	FundingRate          types.Number `json:"r"`
	NextFundingTime      types.Time   `json:"T"`
}

// UFuturesAssetIndexUpdate holds asset index for multi-assets mode user
type UFuturesAssetIndexUpdate struct {
	EventType             string       `json:"e"`
	EventTime             types.Time   `json:"E"`
	Symbol                string       `json:"s"`
	IndexPrice            types.Number `json:"i"`
	BidBuffer             types.Number `json:"b"`
	AskBuffer             types.Number `json:"a"`
	BidRate               types.Number `json:"B"`
	AskRate               types.Number `json:"A"`
	AutoExchangeBidBuffer types.Number `json:"q"`
	AutoExchangeAskbuffer types.Number `json:"g"`
	AutoExchangeBidRate   types.Number `json:"Q"`
	AutoExchangeAskRate   types.Number `json:"G"`
}

// FuturesContractInfo contract info updates. bks field only shows up when bracket gets updated.
type FuturesContractInfo struct {
	EventType        string     `json:"e"`
	EventTime        types.Time `json:"E"`
	Symbol           string     `json:"s"`
	Pair             string     `json:"ps"`
	ContractType     string     `json:"ct"`
	DeliveryDateTime types.Time `json:"dt"`
	OnboardDateTime  types.Time `json:"ot"`
	ContractStatus   string     `json:"cs"`
	Brackets         []struct {
		NationalBracket      float64 `json:"bs"`
		BracketFloorNotional float64 `json:"bnf"`
		BracketNotionalCap   float64 `json:"bnc"`
		MaintenanceRatio     float64 `json:"mmr"`
		Cf                   float64 `json:"cf"`
		MinLeverage          float64 `json:"mi"`
		MaxLeverage          float64 `json:"ma"`
	} `json:"bks"`
}

// MarketLiquidationOrder all Liquidation Order Snapshot Streams push force liquidation order information for all symbols in the market.
type MarketLiquidationOrder struct {
	EventType string     `json:"e"`
	EventTime types.Time `json:"E"`
	Order     struct {
		Symbol                         string       `json:"s"`
		Side                           string       `json:"S"`
		OrderType                      string       `json:"o"`
		TimeInForce                    string       `json:"f"`
		OriginalQuantity               types.Number `json:"q"`
		Price                          types.Number `json:"p"`
		AveragePrice                   types.Number `json:"ap"`
		OrderStatus                    string       `json:"X"`
		OrderLastFieldQuantity         types.Number `json:"l"`
		OrderFilledAccumulatedQuantity types.Number `json:"z"`
		OrderTradeTime                 types.Time   `json:"T"`
	} `json:"o"`
}

// FuturesBookTicker update to the best bid or ask's price or quantity in real-time for a specified symbol.
type FuturesBookTicker struct {
	EventType         string       `json:"e"`
	OrderbookUpdateID int64        `json:"u"`
	EventTime         types.Time   `json:"E"`
	TransactionTime   types.Time   `json:"T"`
	Symbol            string       `json:"s"`
	BestBidPrice      types.Number `json:"b"`
	BestBidQty        types.Number `json:"B"`
	BestAskPrice      types.Number `json:"a"`
	BestAskQty        types.Number `json:"A"`

	// Pair added to coin marigined futures
	Pair string `json:"ps"`
}

// UFutureMarketTicker 24hr rolling window ticker statistics for all symbols.
type UFutureMarketTicker struct {
	EventType             string       `json:"e"`
	EventTime             types.Time   `json:"E"`
	Symbol                string       `json:"s"`
	PriceChange           types.Number `json:"p"`
	PriceChangePercent    types.Number `json:"P"`
	WeightedAveragePrice  types.Number `json:"w"`
	LastPrice             types.Number `json:"c"`
	LastQuantity          types.Number `json:"Q"`
	OpenPrice             types.Number `json:"o"`
	HighPrice             types.Number `json:"h"`
	LowPrice              types.Number `json:"l"`
	TotalTradeBaseVolume  types.Number `json:"v"`
	TotalQuoteAssetVolume types.Number `json:"q"`
	OpenTime              types.Time   `json:"O"`
	CloseTIme             types.Time   `json:"C"`
	FirstTradeID          int64        `json:"F"`
	LastTradeID           int64        `json:"L"`
	TotalNumberOfTrades   int64        `json:"n"`
}

// FutureMiniTickerPrice holds market mini tickers stream
type FutureMiniTickerPrice struct {
	EventType  string       `json:"e"`
	EventTime  types.Time   `json:"E"`
	Symbol     string       `json:"s"`
	ClosePrice types.Number `json:"c"`
	OpenPrice  types.Number `json:"o"`
	HighPrice  types.Number `json:"h"`
	LowPrice   types.Number `json:"l"`
	Volume     types.Number `json:"v"`

	QuoteVolume types.Number `json:"q"` // Total traded base asset volume for Coin Margined Futures

	Pair string `json:"ps"`
}

// FuturesAggTrade aggregate trade streams push market trade
type FuturesAggTrade struct {
	EventType        string       `json:"e"`
	EventTime        types.Time   `json:"E"`
	Symbol           string       `json:"s"`
	AggregateTradeID int64        `json:"a"`
	Price            types.Number `json:"p"`
	Quantity         types.Number `json:"q"`
	FirstTradeID     int64        `json:"f"`
	LastTradeID      int64        `json:"l"`
	TradeTime        types.Time   `json:"T"`
	IsMaker          bool         `json:"m"`
}

// FuturesDepthOrderbook represents bids and asks
type FuturesDepthOrderbook struct {
	EventType               string     `json:"e"`
	EventTime               types.Time `json:"E"`
	TransactionTime         types.Time `json:"T"`
	Symbol                  string     `json:"s"`
	FirstUpdateID           int64      `json:"U"`
	LastUpdateID            int64      `json:"u"`
	FinalUpdateIDLastStream int64      `json:"pu"`
	Bids                    [][]string `json:"b"`
	Asks                    [][]string `json:"a"`

	// Added for coin margined futures
	Pair string `json:"ps"`
}

// UFutureCompositeIndex represents symbols a composite index
type UFutureCompositeIndex struct {
	EventType   string       `json:"e"`
	EventTime   types.Time   `json:"E"`
	Symbol      string       `json:"s"`
	Price       types.Number `json:"p"`
	C           string       `json:"C"`
	Composition []struct {
		BaseAsset          string       `json:"b"`
		QuoteAsset         string       `json:"q"`
		WeightQuantity     types.Number `json:"w"`
		WeightInPercentage types.Number `json:"W"`
		IndexPrice         types.Number `json:"i"`
	} `json:"c"`
}

// FutureContinuousKline represents continuous kline data.
type FutureContinuousKline struct {
	EventType    string     `json:"e"`
	EventTime    types.Time `json:"E"`
	Pair         string     `json:"ps"`
	ContractType string     `json:"ct"`
	KlineData    struct {
		StartTime                types.Time   `json:"t"`
		EndTime                  types.Time   `json:"T"`
		Interval                 string       `json:"i"`
		FirstUpdateID            int64        `json:"f"`
		LastupdateID             int64        `json:"L"`
		OpenPrice                types.Number `json:"o"`
		ClosePrice               types.Number `json:"c"`
		HighPrice                types.Number `json:"h"`
		LowPrice                 types.Number `json:"l"`
		Volume                   types.Number `json:"v"`
		NumberOfTrades           int64        `json:"n"`
		IsKlineClosed            bool         `json:"x"`
		QuoteAssetVolume         types.Number `json:"q"`
		TakerBuyVolume           types.Number `json:"V"`
		TakerBuyQuoteAssetVolume types.Number `json:"Q"`
		B                        string       `json:"B"`
	} `json:"k"`
}

// WebsocketActionResponse represents a response for websocket actions like "SET_PROPERTY", "LIST_SUBSCRIPTIONS" and others
type WebsocketActionResponse struct {
	Result []string `json:"result"`
	ID     int64    `json:"id"`
}

// RateLimitItem holds ratelimit information for endpoint calls.
type RateLimitItem struct {
	RateLimitType  string `json:"rateLimitType"`
	Interval       string `json:"interval"`
	IntervalNumber int64  `json:"intervalNum"`
	Limit          int64  `json:"limit"`
	Count          int64  `json:"count"`
}

// SymbolAveragePrice represents the average symbol price
type SymbolAveragePrice struct {
	PriceIntervalMins int64        `json:"mins"`
	Price             types.Number `json:"price"`
	CloseTime         types.Time   `json:"closeTime"`
}

// PriceChangeRequestParam holds request parameters for price change request parameters
type PriceChangeRequestParam struct {
	Symbol     string          `json:"symbol,omitempty"`
	Symbols    []currency.Pair `json:"symbols,omitempty"`
	Timezone   string          `json:"timeZone,omitempty"`
	TickerType string          `json:"type,omitempty"`
}

// PriceChanges holds a single or slice of WsTickerPriceChange instance into a new type.
type PriceChanges []PriceChangeStats

// WsRollingWindowPriceParams rolling window price change statistics request params
type WsRollingWindowPriceParams struct {
	Symbols            []currency.Pair `json:"symbols,omitempty"`
	WindowSizeDuration time.Duration   `json:"-"`
	WindowSize         string          `json:"windowSize,omitempty"`
	TickerType         string          `json:"type,omitempty"`
	Symbol             string          `json:"symbol,omitempty"`
}

// SymbolTickerItem holds symbol and price information
type SymbolTickerItem struct {
	Symbol string       `json:"symbol"`
	Price  types.Number `json:"price"`
}

// SymbolTickers holds symbol and price ticker information.
type SymbolTickers []SymbolTickerItem

// WsOrderbookTicker holds orderbook ticker information
type WsOrderbookTicker struct {
	Symbol   string       `json:"symbol"`
	BidPrice types.Number `json:"bidPrice"`
	BidQty   types.Number `json:"bidQty"`
	AskPrice types.Number `json:"askPrice"`
	AskQty   types.Number `json:"askQty"`
}

// WsOrderbookTickers represents an orderbook ticker information
type WsOrderbookTickers []WsOrderbookTicker

// APISignatureInfo holds API key and signature information
type APISignatureInfo struct {
	APIKey    string `json:"apiKey,omitempty"`
	Signature string `json:"signature,omitempty"`
	Timestamp int64  `json:"timestamp,omitempty"`
}

// TradeOrderRequestParam new order request parameter
type TradeOrderRequestParam struct {
	APISignatureInfo
	Symbol      string  `json:"symbol"`
	Side        string  `json:"side"`
	OrderType   string  `json:"type"`
	TimeInForce string  `json:"timeInForce"`
	Price       float64 `json:"price,omitempty,string"`
	Quantity    float64 `json:"quantity,omitempty,string"`
}

// QueryOrderParam represents an order querying parameters
type QueryOrderParam struct {
	APISignatureInfo
	Symbol            string `json:"symbol,omitempty"`
	OrderID           int64  `json:"orderId,omitempty"`
	OrigClientOrderID string `json:"origClientOrderId,omitempty"`
	RecvWindow        int64  `json:"recvWindow,omitempty"`

	NewClientOrderID   string `json:"newClientOrderId,omitempty"`
	CancelRestrictions string `json:"cancelRestrictions,omitempty"`
}

// WsCancelAndReplaceParam represents a cancel and replace request parameters
type WsCancelAndReplaceParam struct {
	APISignatureInfo
	Symbol        string `json:"symbol,omitempty"`
	CancelOrderID string `json:"cancelOrderId,omitempty"`

	// CancelReplaceMode possible values are 'STOP_ON_FAILURE', 'ALLOW_FAILURE'
	CancelReplaceMode         string  `json:"cancelReplaceMode,omitempty"`
	CancelNewClientOrderID    string  `json:"cancelNewClientOrderId,omitempty"`
	CancelOriginClientOrderID string  `json:"cancelOrigClientOrderId,omitempty"`
	Side                      string  `json:"side,omitempty"` // BUY or SELL
	Price                     float64 `json:"price,omitempty"`
	Quantity                  float64 `json:"quantity,omitempty"`
	OrderType                 string  `json:"type,omitempty"`
	TimeInForce               string  `json:"timeInForce,omitempty"`
	QuoteOrderQty             float64 `json:"quoteOrderQty,omitempty"`
	NewClientOrderID          string  `json:"newClientOrderId,omitempty"`

	// Select response format: ACK, RESULT, FULL.
	NewOrderRespType string  `json:"newOrderRespType,omitempty"` // Select response format: ACK, RESULT, FULL. MARKET and LIMIT orders produce FULL response by default, other order types default to ACK.
	StopPrice        float64 `json:"stopPrice,omitempty"`
	TrailingDelta    float64 `json:"trailingDelta,omitempty"`
	IcebergQty       float64 `json:"icebergQty,omitempty"`
	StrategyID       int64   `json:"strategyId,omitempty"`

	// Values smaller than 1000000 are reserved and cannot be used.
	StrategyType int64 `json:"strategyType,omitempty"`

	// The possible supported values are EXPIRE_TAKER, EXPIRE_MAKER, EXPIRE_BOTH, NONE.
	SelfTradePreventionMode string `json:"selfTradePreventionMode,omitempty"`

	// Supported values:
	// ONLY_NEW - Cancel will succeed if the order status is NEW.
	// ONLY_PARTIALLY_FILLED - Cancel will succeed if order status is PARTIALLY_FILLED. For more information please refer to Regarding cancelRestrictions.
	CancelRestrictions string `json:"cancelRestrictions,omitempty"`
	RecvWindow         int64  `json:"recvWindow,omitempty"`
}

// PlaceOCOOrderParam holds a request parameters for one-cancel-other orders
type PlaceOCOOrderParam struct {
	APISignatureInfo
	Symbol               string  `json:"symbol,omitempty"`
	Side                 string  `json:"side,omitempty"`
	Price                float64 `json:"price,omitempty"`
	Quantity             float64 `json:"quantity,omitempty"`
	ListClientOrderID    string  `json:"listClientOrderId,omitempty"`
	LimitClientOrderID   string  `json:"limitClientOrderId,omitempty"`
	LimitIcebergQty      float64 `json:"limitIcebergQty,omitempty"`
	LimitStrategyID      string  `json:"limitStrategyId,omitempty"`
	LimitStrategyType    string  `json:"limitStrategyType,omitempty"`
	StopPrice            float64 `json:"stopPrice,omitempty"`
	TrailingDelta        int64   `json:"trailingDelta,omitempty"`
	StopClientOrderID    string  `json:"stopClientOrderId,omitempty"`
	StopLimitPrice       float64 `json:"stopLimitPrice,omitempty"`
	StopLimitTimeInForce string  `json:"stopLimitTimeInForce,omitempty"`
	StopIcebergQty       float64 `json:"stopIcebergQty,omitempty"`
	StopStrategyID       string  `json:"stopStrategyId,omitempty"`
	StopStrategyType     string  `json:"stopStrategyType,omitempty"`
	NewOrderRespType     string  `json:"newOrderRespType,omitempty"`

	// The allowed enums is dependent on what is configured on the symbol. The possible supported values are 'EXPIRE_TAKER', 'EXPIRE_MAKER', 'EXPIRE_BOTH', 'NONE'.
	SelfTradePreventionMode string `json:"selfTradePreventionMode,omitempty"`
	RecvWindow              int64  `json:"recvWindow,omitempty"`
}

// TradeOrderResponse holds response for trade order.
type TradeOrderResponse struct {
	Symbol        string     `json:"symbol"`
	OrderID       int64      `json:"orderId"`
	OrderListID   int64      `json:"orderListId"`
	ClientOrderID string     `json:"clientOrderId"`
	TransactTime  types.Time `json:"transactTime"`
}

// FuturesAuthenticationResp holds authentication.
type FuturesAuthenticationResp struct {
	APIKey           string     `json:"apiKey"`
	AuthorizedSince  int64      `json:"authorizedSince"`
	ConnectedSince   int64      `json:"connectedSince"`
	ReturnRateLimits bool       `json:"returnRateLimits"`
	ServerTime       types.Time `json:"serverTime"`
}

// WsCancelAndReplaceTradeOrderResponse holds a response from cancel and replacing an existing trade order
type WsCancelAndReplaceTradeOrderResponse struct {
	CancelResult   string `json:"cancelResult"`
	NewOrderResult string `json:"newOrderResult"`
	CancelResponse struct {
		Symbol                  string       `json:"symbol"`
		OrigClientOrderID       string       `json:"origClientOrderId"`
		OrderID                 int64        `json:"orderId"`
		OrderListID             int64        `json:"orderListId"`
		ClientOrderID           string       `json:"clientOrderId"`
		TransactTime            types.Time   `json:"transactTime"`
		Price                   types.Number `json:"price"`
		OrigQty                 types.Number `json:"origQty"`
		ExecutedQty             types.Number `json:"executedQty"`
		CummulativeQuoteQty     types.Number `json:"cummulativeQuoteQty"`
		Status                  string       `json:"status"`
		TimeInForce             string       `json:"timeInForce"`
		Type                    string       `json:"type"`
		Side                    string       `json:"side"`
		SelfTradePreventionMode string       `json:"selfTradePreventionMode"`
	} `json:"cancelResponse"`
	NewOrderResponse struct {
		Symbol                  string       `json:"symbol"`
		OrderID                 int64        `json:"orderId"`
		OrderListID             int64        `json:"orderListId"`
		ClientOrderID           string       `json:"clientOrderId"`
		TransactTime            types.Time   `json:"transactTime"`
		Price                   types.Number `json:"price"`
		OrigQty                 types.Number `json:"origQty"`
		ExecutedQty             types.Number `json:"executedQty"`
		CummulativeQuoteQty     types.Number `json:"cummulativeQuoteQty"`
		Status                  string       `json:"status"`
		TimeInForce             string       `json:"timeInForce"`
		Type                    string       `json:"type"`
		Side                    string       `json:"side"`
		WorkingTime             types.Time   `json:"workingTime"`
		Fills                   []any        `json:"fills"`
		SelfTradePreventionMode string       `json:"selfTradePreventionMode"`
	} `json:"newOrderResponse"`
}

// WsCancelOrder holds a response data for canceling an open order.
type WsCancelOrder struct {
	Symbol                  string       `json:"symbol"`
	OrigClientOrderID       string       `json:"origClientOrderId,omitempty"`
	OrderID                 int64        `json:"orderId,omitempty"`
	OrderListID             int64        `json:"orderListId"`
	ClientOrderID           string       `json:"clientOrderId,omitempty"`
	TransactTime            types.Time   `json:"transactTime,omitempty"`
	Price                   types.Number `json:"price,omitempty"`
	OrigQty                 types.Number `json:"origQty,omitempty"`
	ExecutedQty             types.Number `json:"executedQty,omitempty"`
	CummulativeQuoteQty     types.Number `json:"cummulativeQuoteQty,omitempty"`
	Status                  string       `json:"status,omitempty"`
	TimeInForce             string       `json:"timeInForce,omitempty"`
	Type                    string       `json:"type,omitempty"`
	Side                    string       `json:"side,omitempty"`
	StopPrice               types.Number `json:"stopPrice,omitempty"`
	IcebergQty              types.Number `json:"icebergQty,omitempty"`
	StrategyID              int64        `json:"strategyId,omitempty"`
	StrategyType            int64        `json:"strategyType,omitempty"`
	SelfTradePreventionMode string       `json:"selfTradePreventionMode,omitempty"`
	ContingencyType         string       `json:"contingencyType,omitempty"`
	ListStatusType          string       `json:"listStatusType,omitempty"`
	ListOrderStatus         string       `json:"listOrderStatus,omitempty"`
	ListClientOrderID       string       `json:"listClientOrderId,omitempty"`
	TransactionTime         types.Time   `json:"transactionTime,omitempty"`
	Orders                  []struct {
		Symbol        string `json:"symbol"`
		OrderID       int64  `json:"orderId"`
		ClientOrderID string `json:"clientOrderId"`
	} `json:"orders,omitempty"`
	OrderReports []OrderReportItem `json:"orderReports,omitempty"`
}

// OrderReportItem represents a single order report instance.
type OrderReportItem struct {
	Symbol                  string       `json:"symbol"`
	OrigClientOrderID       string       `json:"origClientOrderId"`
	OrderID                 int64        `json:"orderId"`
	OrderListID             int64        `json:"orderListId"`
	ClientOrderID           string       `json:"clientOrderId"`
	TransactTime            types.Time   `json:"transactTime"`
	Price                   types.Number `json:"price"`
	OrigQty                 types.Number `json:"origQty"`
	ExecutedQty             types.Number `json:"executedQty"`
	CummulativeQuoteQty     types.Number `json:"cummulativeQuoteQty"`
	Status                  string       `json:"status"`
	TimeInForce             string       `json:"timeInForce"`
	Type                    string       `json:"type"`
	Side                    string       `json:"side"`
	StopPrice               types.Number `json:"stopPrice,omitempty"`
	SelfTradePreventionMode string       `json:"selfTradePreventionMode"`
}

// OCOOrder represents a one-close-other order type.
type OCOOrder struct {
	OrderListID       int64      `json:"orderListId"`
	ContingencyType   string     `json:"contingencyType"`
	ListStatusType    string     `json:"listStatusType"`
	ListOrderStatus   string     `json:"listOrderStatus"`
	ListClientOrderID string     `json:"listClientOrderId"`
	TransactionTime   types.Time `json:"transactionTime"`
	Symbol            string     `json:"symbol"`
	Orders            []struct {
		Symbol        string `json:"symbol"`
		OrderID       int64  `json:"orderId"`
		ClientOrderID string `json:"clientOrderId"`
	} `json:"orders"`
	OrderReports []struct {
		Symbol                  string       `json:"symbol"`
		OrderID                 int64        `json:"orderId"`
		OrderListID             int64        `json:"orderListId"`
		ClientOrderID           string       `json:"clientOrderId"`
		TransactTime            types.Time   `json:"transactTime"`
		Price                   types.Number `json:"price"`
		OrigQty                 types.Number `json:"origQty"`
		ExecutedQty             types.Number `json:"executedQty"`
		CummulativeQuoteQty     types.Number `json:"cummulativeQuoteQty"`
		Status                  string       `json:"status"`
		TimeInForce             string       `json:"timeInForce"`
		Type                    string       `json:"type"`
		Side                    string       `json:"side"`
		StopPrice               types.Number `json:"stopPrice,omitempty"`
		WorkingTime             types.Time   `json:"workingTime"`
		SelfTradePreventionMode string       `json:"selfTradePreventionMode"`

		OrigClientOrderID string `json:"origClientOrderId"`
	} `json:"orderReports"`

	MarginBuyBorrowAmount types.Number `json:"marginBuyBorrowAmount"`
	MarginBuyBorrowAsset  string       `json:"marginBuyBorrowAsset"`
}

// OCOOrderInfo represents OCO order information.
type OCOOrderInfo struct {
	OrderListID       int64      `json:"orderListId"`
	ContingencyType   string     `json:"contingencyType"`
	ListStatusType    string     `json:"listStatusType"`
	ListOrderStatus   string     `json:"listOrderStatus"`
	ListClientOrderID string     `json:"listClientOrderId"`
	TransactionTime   types.Time `json:"transactionTime"`
	Symbol            string     `json:"symbol"`
	Orders            []struct {
		Symbol        string `json:"symbol"`
		OrderID       int64  `json:"orderId"`
		ClientOrderID string `json:"clientOrderId"`
	} `json:"orders"`

	// returned when cancelling the order
	OrderReports []struct {
		Symbol                  string       `json:"symbol"`
		OrderID                 int64        `json:"orderId"`
		OrderListID             int64        `json:"orderListId"`
		ClientOrderID           string       `json:"clientOrderId"`
		TransactTime            types.Time   `json:"transactTime"`
		Price                   types.Number `json:"price"`
		OrigQty                 types.Number `json:"origQty"`
		ExecutedQty             types.Number `json:"executedQty"`
		CummulativeQuoteQty     types.Number `json:"cummulativeQuoteQty"`
		Status                  string       `json:"status"`
		TimeInForce             string       `json:"timeInForce"`
		Type                    string       `json:"type"`
		Side                    string       `json:"side"`
		StopPrice               types.Number `json:"stopPrice,omitempty"`
		SelfTradePreventionMode string       `json:"selfTradePreventionMode"`
	} `json:"orderReports"`
}

// WsOSRPlaceOrderParams holds request parameters for placing OSR orders.
type WsOSRPlaceOrderParams struct {
	APISignatureInfo
	Symbol           string  `json:"symbol,omitempty"`
	Side             string  `json:"side,omitempty"`
	OrderType        string  `json:"type,omitempty"`
	TimeInForce      string  `json:"timeInForce,omitempty"`
	Price            float64 `json:"price,omitempty"`
	Quantity         float64 `json:"quantity,omitempty"`
	NewClientOrderID string  `json:"newClientOrderId,omitempty"`

	// Select response format: ACK, RESULT, FULL.
	// MARKET and LIMIT orders use FULL by default.
	NewOrderRespType string  `json:"newOrderRespType,omitempty"`
	IcebergQty       float64 `json:"icebergQty,omitempty"`
	StrategyID       int64   `json:"strategyId,omitempty"`
	StrategyType     string  `json:"strategyType,omitempty"`

	// The allowed enums is dependent on what is configured on the symbol. The possible supported values are EXPIRE_TAKER, EXPIRE_MAKER, EXPIRE_BOTH, NONE.
	SelfTradePreventionMode string `json:"selfTradePreventionMode,omitempty"`
	RecvWindow              string `json:"recvWindow,omitempty"`
}

// OSROrder represents a request parameters for Smart Order Routing (SOR)
type OSROrder struct {
	Symbol              string       `json:"symbol"`
	OrderID             int64        `json:"orderId"`
	OrderListID         int64        `json:"orderListId"`
	ClientOrderID       string       `json:"clientOrderId"`
	TransactTime        types.Time   `json:"transactTime"`
	Price               types.Number `json:"price"`
	OrigQty             types.Number `json:"origQty"`
	ExecutedQty         types.Number `json:"executedQty"`
	CummulativeQuoteQty types.Number `json:"cummulativeQuoteQty"`
	Status              string       `json:"status"`
	TimeInForce         string       `json:"timeInForce"`
	Type                string       `json:"type"`
	Side                string       `json:"side"`
	WorkingTime         types.Time   `json:"workingTime"`
	Fills               []struct {
		MatchType       string       `json:"matchType"`
		Price           types.Number `json:"price"`
		Qty             types.Number `json:"qty"`
		Commission      string       `json:"commission"`
		CommissionAsset string       `json:"commissionAsset"`
		TradeID         int64        `json:"tradeId"`
		AllocID         int64        `json:"allocId"`
	} `json:"fills"`
	WorkingFloor            string `json:"workingFloor"`
	SelfTradePreventionMode string `json:"selfTradePreventionMode"`
	UsedSor                 bool   `json:"usedSor"`
}

// AccountOrderRequestParam retrieves an account order history parameters
type AccountOrderRequestParam struct {
	APISignatureInfo
	EndTime    int64  `json:"endTime,omitempty"`
	Limit      int64  `json:"limit,omitempty"`
	OrderID    int64  `json:"orderId,omitempty"` // Order ID to begin at
	RecvWindow int64  `json:"recvWindow,omitempty"`
	StartTime  int64  `json:"startTime,omitempty"`
	Symbol     string `json:"symbol"`

	// for requesting trades
	FromID int64 `json:"fromId,omitempty"`
}

// TradeHistory holds trade history information.
type TradeHistory struct {
	Symbol          string       `json:"symbol"`
	ID              int          `json:"id"`
	OrderID         int64        `json:"orderId"`
	OrderListID     int          `json:"orderListId"`
	Price           types.Number `json:"price"`
	Qty             types.Number `json:"qty"`
	QuoteQty        types.Number `json:"quoteQty"`
	Commission      types.Number `json:"commission"`
	CommissionAsset string       `json:"commissionAsset"`
	Time            types.Time   `json:"time"`
	IsBuyer         bool         `json:"isBuyer"`
	IsMaker         bool         `json:"isMaker"`
	IsBestMatch     bool         `json:"isBestMatch"`
	IsIsolated      bool         `json:"isIsolated"` // added for margin accounts trade list information
}

// SelfTradePrevention represents a self-trade prevention instance.
type SelfTradePrevention struct {
	Symbol                  string       `json:"symbol"`
	PreventedMatchID        int64        `json:"preventedMatchId"`
	TakerOrderID            int64        `json:"takerOrderId"`
	MakerOrderID            int64        `json:"makerOrderId"`
	TradeGroupID            int64        `json:"tradeGroupId"`
	SelfTradePreventionMode string       `json:"selfTradePreventionMode"`
	Price                   types.Number `json:"price"`
	MakerPreventedQuantity  types.Number `json:"makerPreventedQuantity"`
	TransactTime            types.Time   `json:"transactTime"`
}

// SORReplacements represents response instance after for Smart Order Routing(SOR) order placement.
type SORReplacements struct {
	Symbol          string       `json:"symbol"`
	AllocationID    int64        `json:"allocationId"`
	AllocationType  string       `json:"allocationType"`
	OrderID         int64        `json:"orderId"`
	OrderListID     int64        `json:"orderListId"`
	Price           types.Number `json:"price"`
	Quantity        types.Number `json:"qty"`
	QuoteQty        types.Number `json:"quoteQty"`
	Commission      string       `json:"commission"`
	CommissionAsset string       `json:"commissionAsset"`
	Time            types.Time   `json:"time"`
	IsBuyer         bool         `json:"isBuyer"`
	IsMaker         bool         `json:"isMaker"`
	IsAllocator     bool         `json:"isAllocator"`
}

// CommissionRateInto represents commission rate info.
type CommissionRateInto struct {
	Symbol             string          `json:"symbol"`
	StandardCommission *CommissionInfo `json:"standardCommission"`
	TaxCommission      *CommissionInfo `json:"taxCommission"`
	Discount           struct {
		EnabledForAccount bool   `json:"enabledForAccount"`
		EnabledForSymbol  bool   `json:"enabledForSymbol"`
		DiscountAsset     string `json:"discountAsset"`
		Discount          string `json:"discount"`
	} `json:"discount"`
}

// CommissionInfo holds tax and standard
type CommissionInfo struct {
	Maker  string `json:"maker"`
	Taker  string `json:"taker"`
	Buyer  string `json:"buyer"`
	Seller string `json:"seller"`
}

// SystemStatus holds system status code and message
type SystemStatus struct {
	Status  int64  `json:"status"` // 0: normal，1：system maintenance
	Message string `json:"msg"`    // "normal", "system_maintenance"
}

// DailyAccountSnapshot holds a snapshot of daily asset information
type DailyAccountSnapshot struct {
	Code        int64  `json:"code"`
	Msg         string `json:"msg"`
	SnapshotVos []struct {
		Data struct {
			Balances []struct {
				Asset  string       `json:"asset"`
				Free   types.Number `json:"free"`
				Locked types.Number `json:"locked"`
			} `json:"balances"`
			TotalAssetOfBTC string `json:"totalAssetOfBtc"`
		} `json:"data"`
		Type       string     `json:"type"`
		UpdateTime types.Time `json:"updateTime"`
	} `json:"snapshotVos"`
}

// TradingAPIAccountStatus represents a trading account
type TradingAPIAccountStatus struct {
	Data struct {
		IsLocked           bool       `json:"isLocked"`
		PlannedRecoverTime types.Time `json:"plannedRecoverTime"`
		TriggerCondition   struct {
			Gcr  int64 `json:"GCR"`
			Ifer int64 `json:"IFER"`
			Ufr  int64 `json:"UFR"`
		} `json:"triggerCondition"`
		UpdateTime types.Time `json:"updateTime"`
	} `json:"data"`
}

// DustLog holds small assets information
type DustLog struct {
	Total              int64 `json:"total"`
	UserAssetDribblets []struct {
		OperateTime              types.Time   `json:"operateTime"`
		TotalTransferedAmount    types.Number `json:"totalTransferedAmount"`
		TotalServiceChargeAmount types.Number `json:"totalServiceChargeAmount"`
		TransID                  int64        `json:"transId"`
		UserAssetDribbletDetails []struct {
			TransID             int64        `json:"transId"`
			ServiceChargeAmount types.Number `json:"serviceChargeAmount"`
			Amount              types.Number `json:"amount"`
			OperateTime         types.Time   `json:"operateTime"`
			TransferedAmount    types.Number `json:"transferedAmount"`
			FromAsset           string       `json:"fromAsset"`
		} `json:"userAssetDribbletDetails"`
	} `json:"userAssetDribblets"`
}

// DepositAddressAndNetwork represents a deposit address with network
type DepositAddressAndNetwork struct {
	Coin      string `json:"coin"`
	Address   string `json:"address"`
	IsDefault int64  `json:"isDefault"`
}

// UserWalletBalance represents a user wallet balance information.
type UserWalletBalance struct {
	Activate   bool         `json:"activate"`
	Balance    types.Number `json:"balance"`
	WalletName string       `json:"walletName"`
}

// UserDelegationHistory represents a user delegation history
type UserDelegationHistory struct {
	Total int64 `json:"total"`
	Rows  []struct {
		ClientTranID string       `json:"clientTranId"`
		TransferType string       `json:"transferType"`
		Asset        string       `json:"asset"`
		Amount       types.Number `json:"amount"`
		Time         types.Time   `json:"time"`
	} `json:"rows"`
}

// DelistSchedule symbols delist schedule for spot
type DelistSchedule struct {
	DelistTime types.Time `json:"delistTime"`
	Symbols    []string   `json:"symbols"`
}

// WithdrawAddress represents a withdraw address item detail.
type WithdrawAddress struct {
	Address     string `json:"address"`
	AddressTag  string `json:"addressTag"`
	Coin        string `json:"coin"`
	Name        string `json:"name"` // is a user-defined name
	Network     string `json:"network"`
	Origin      string `json:"origin"`      // if originType=='others', the address source manually filled in by the user
	OriginType  string `json:"originType"`  // Address source type
	WhiteStatus bool   `json:"whiteStatus"` // Is it whitelisted
}

// VirtualSubAccount represents a response information after creating the virtual account.
type VirtualSubAccount struct {
	Email string `json:"email"`
}

// SubAccountList represents a response
type SubAccountList struct {
	SubAccounts []struct {
		Email                       string     `json:"email"`
		IsFreeze                    bool       `json:"isFreeze"`
		CreateTime                  types.Time `json:"createTime"`
		IsManagedSubAccount         bool       `json:"isManagedSubAccount"`
		IsAssetManagementSubAccount bool       `json:"isAssetManagementSubAccount"`
	} `json:"subAccounts"`
}

// SubAccountSpotAsset represents a spot asset transfer item
type SubAccountSpotAsset struct {
	From   string       `json:"from"`
	To     string       `json:"to"`
	Asset  string       `json:"asset"`
	Qty    types.Number `json:"qty"`
	Status string       `json:"status"`
	TranID int64        `json:"tranId"`
	Time   types.Time   `json:"time"`
}

// AssetTransferHistory Query Sub-account Futures Asset Transfer History For Master Account
type AssetTransferHistory struct {
	Success     bool  `json:"success"`
	FuturesType int64 `json:"futuresType"`
	Transfers   []struct {
		From   string       `json:"from"`
		To     string       `json:"to"`
		Asset  string       `json:"asset"`
		Qty    types.Number `json:"qty"`
		TranID int64        `json:"tranId"`
		Time   types.Time   `json:"time"`
	} `json:"transfers"`
}

// FuturesAssetTransfer represents a futures asset transfer response.
type FuturesAssetTransfer struct {
	Success       bool   `json:"success"`
	TransactionID string `json:"txnId"`
}

// SubAccountAssets represents a sub-account asset
type SubAccountAssets struct {
	Balances []struct {
		Asset  string  `json:"asset"`
		Free   float64 `json:"free"`
		Locked float64 `json:"locked"`
	} `json:"balances"`
}

// SubAccountSpotSummary asset summary of subaccounts.
type SubAccountSpotSummary struct {
	TotalCount                int64  `json:"totalCount"`
	MasterAccountTotalAsset   string `json:"masterAccountTotalAsset"`
	SpotSubUserAssetBtcVoList []struct {
		Email      string       `json:"email"`
		TotalAsset types.Number `json:"totalAsset"`
	} `json:"spotSubUserAssetBtcVoList"`
}

// SubAccountDepositAddress represents a sub-acccount deposit address for master account
type SubAccountDepositAddress struct {
	Address string `json:"address"`
	Coin    string `json:"coin"`
	Tag     string `json:"tag"`
	URL     string `json:"url"`
}

// SubAccountDepositHistory represents a sub account deposit history
type SubAccountDepositHistory struct {
	ID            string       `json:"id"`
	Amount        types.Number `json:"amount"`
	Coin          string       `json:"coin"`
	Network       string       `json:"network"`
	Status        int64        `json:"status"`
	Address       string       `json:"address"`
	AddressTag    string       `json:"addressTag"`
	TxID          string       `json:"txId"`
	InsertTime    types.Time   `json:"insertTime"`
	TransferType  int64        `json:"transferType"`
	ConfirmTimes  string       `json:"confirmTimes"`
	UnlockConfirm int64        `json:"unlockConfirm"`
	WalletType    int64        `json:"walletType"`
}

// SubAccountStatus represents sub-account status on margin/futures for master account.
type SubAccountStatus struct {
	Email            string     `json:"email"`
	IsSubUserEnabled bool       `json:"isSubUserEnabled"`
	IsUserActive     bool       `json:"isUserActive"`
	InsertTime       types.Time `json:"insertTime"`
	IsMarginEnabled  bool       `json:"isMarginEnabled"`
	IsFutureEnabled  bool       `json:"isFutureEnabled"`
	Mobile           int64      `json:"mobile"`
}

// MarginEnablingResponse represents a Margin sub-account for master account
type MarginEnablingResponse struct {
	Email           string `json:"email"`
	IsMarginEnabled bool   `json:"isMarginEnabled"`
}

// SubAccountMarginAccountDetail represents a sub-account margin account detail.
type SubAccountMarginAccountDetail struct {
	Email               string       `json:"email"`
	MarginLevel         types.Number `json:"marginLevel"`
	TotalAssetOfBtc     types.Number `json:"totalAssetOfBtc"`
	TotalLiabilityOfBtc types.Number `json:"totalLiabilityOfBtc"`
	TotalNetAssetOfBtc  types.Number `json:"totalNetAssetOfBtc"`
	MarginTradeCoeffVo  struct {
		ForceLiquidationBar types.Number `json:"forceLiquidationBar"`
		MarginCallBar       types.Number `json:"marginCallBar"`
		NormalBar           types.Number `json:"normalBar"`
	} `json:"marginTradeCoeffVo"`
	MarginUserAssetVoList []struct {
		Asset    string       `json:"asset"`
		Borrowed types.Number `json:"borrowed"`
		Free     types.Number `json:"free"`
		Interest types.Number `json:"interest"`
		Locked   types.Number `json:"locked"`
		NetAsset types.Number `json:"netAsset"`
	} `json:"marginUserAssetVoList"`
}

// SubAccountMarginAccount represents a sub-account margin detail.
type SubAccountMarginAccount struct {
	TotalAssetOfBTC     string `json:"totalAssetOfBtc"`
	TotalLiabilityOfBTC string `json:"totalLiabilityOfBtc"`
	TotalNetAssetOfBTC  string `json:"totalNetAssetOfBtc"`
	SubAccountList      []struct {
		Email               string `json:"email"`
		TotalAssetOfBTC     string `json:"totalAssetOfBtc"`
		TotalLiabilityOfBTC string `json:"totalLiabilityOfBtc"`
		TotalNetAssetOfBTC  string `json:"totalNetAssetOfBtc"`
	} `json:"subAccountList"`
}

// FuturesEnablingResponse represents a futures enabling response.
type FuturesEnablingResponse struct {
	Email            string `json:"email"`
	IsFuturesEnabled bool   `json:"isFuturesEnabled"`
}

// SubAccountsFuturesAccount respresnts futures account for sub accounts.
type SubAccountsFuturesAccount struct {
	Email  string `json:"email"`
	Asset  string `json:"asset"`
	Assets []struct {
		Asset                  string       `json:"asset"`
		InitialMargin          types.Number `json:"initialMargin"`
		MaintenanceMargin      types.Number `json:"maintenanceMargin"`
		MarginBalance          types.Number `json:"marginBalance"`
		MaxWithdrawAmount      types.Number `json:"maxWithdrawAmount"`
		OpenOrderInitialMargin types.Number `json:"openOrderInitialMargin"`
		PositionInitialMargin  types.Number `json:"positionInitialMargin"`
		UnrealizedProfit       string       `json:"unrealizedProfit"`
		WalletBalance          types.Number `json:"walletBalance"`
	} `json:"assets"`
	CanDeposit                  bool         `json:"canDeposit"`
	CanTrade                    bool         `json:"canTrade"`
	CanWithdraw                 bool         `json:"canWithdraw"`
	FeeTier                     int64        `json:"feeTier"`
	MaxWithdrawAmount           types.Number `json:"maxWithdrawAmount"`
	TotalInitialMargin          types.Number `json:"totalInitialMargin"`
	TotalMaintenanceMargin      types.Number `json:"totalMaintenanceMargin"`
	TotalMarginBalance          types.Number `json:"totalMarginBalance"`
	TotalOpenOrderInitialMargin types.Number `json:"totalOpenOrderInitialMargin"`
	TotalPositionInitialMargin  types.Number `json:"totalPositionInitialMargin"`
	TotalUnrealizedProfit       types.Number `json:"totalUnrealizedProfit"`
	TotalWalletBalance          types.Number `json:"totalWalletBalance"`
	UpdateTime                  types.Time   `json:"updateTime"`
}

// SubAccountFuturesAccountSummary represents a sub-account's futures account summary information
type SubAccountFuturesAccountSummary struct {
	TotalInitialMargin          types.Number `json:"totalInitialMargin"`
	TotalMaintenanceMargin      types.Number `json:"totalMaintenanceMargin"`
	TotalMarginBalance          types.Number `json:"totalMarginBalance"`
	TotalOpenOrderInitialMargin types.Number `json:"totalOpenOrderInitialMargin"`
	TotalPositionInitialMargin  types.Number `json:"totalPositionInitialMargin"`
	TotalUnrealizedProfit       types.Number `json:"totalUnrealizedProfit"`
	TotalWalletBalance          types.Number `json:"totalWalletBalance"`
	Asset                       string       `json:"asset"`
	SubAccountList              []struct {
		Email                       string       `json:"email"`
		TotalInitialMargin          types.Number `json:"totalInitialMargin"`
		TotalMaintenanceMargin      types.Number `json:"totalMaintenanceMargin"`
		TotalMarginBalance          types.Number `json:"totalMarginBalance"`
		TotalOpenOrderInitialMargin types.Number `json:"totalOpenOrderInitialMargin"`
		TotalPositionInitialMargin  types.Number `json:"totalPositionInitialMargin"`
		TotalUnrealizedProfit       types.Number `json:"totalUnrealizedProfit"`
		TotalWalletBalance          types.Number `json:"totalWalletBalance"`
		Asset                       string       `json:"asset"`
	} `json:"subAccountList"`
}

// SubAccountFuturesPositionRisk represents futures Position-Risk of sub-account for master account
type SubAccountFuturesPositionRisk struct {
	EntryPrice       types.Number `json:"entryPrice"`
	Leverage         types.Number `json:"leverage"`
	MaxNotional      types.Number `json:"maxNotional"`
	LiquidationPrice types.Number `json:"liquidationPrice"`
	MarkPrice        types.Number `json:"markPrice"`
	PositionAmount   types.Number `json:"positionAmount"`
	Symbol           string       `json:"symbol"`
	UnrealizedProfit types.Number `json:"unrealizedProfit"`
}

// SubAccountTransferHistory represents a subaccount transfer history
type SubAccountTransferHistory struct {
	CounterParty    string       `json:"counterParty"`
	Email           string       `json:"email"`
	Type            int64        `json:"type"`
	Asset           string       `json:"asset"`
	Quantity        types.Number `json:"qty"`
	FromAccountType string       `json:"fromAccountType"`
	ToAccountType   string       `json:"toAccountType"`
	Status          string       `json:"status"`
	TransactionID   int64        `json:"tranId"`
	Time            types.Time   `json:"time"`
}

// SubAccountTransferHistoryItem represents aub-account transfer history from sub-accounts
type SubAccountTransferHistoryItem struct {
	CounterParty    string     `json:"counterParty"`
	Email           string     `json:"email"`
	Type            int64      `json:"type"`
	Asset           string     `json:"asset"`
	Qty             string     `json:"qty"`
	FromAccountType string     `json:"fromAccountType"`
	ToAccountType   string     `json:"toAccountType"`
	Status          string     `json:"status"`
	TranID          int64      `json:"tranId"`
	Time            types.Time `json:"time"`
}

// UniversalTransferParams represents a universal transfer parameters.
type UniversalTransferParams struct {
	FromEmail           string        `json:"fromEmail,omitempty"`
	ToEmail             string        `json:"toEmail,omitempty"`
	FromAccountType     string        `json:"fromAccountType"`
	ToAccountType       string        `json:"toAccountType"`
	ClientTransactionID string        `json:"clientTranId,omitempty"`
	Symbol              string        `json:"symbol,omitempty"`
	Asset               currency.Code `json:"asset"`
	Amount              float64       `json:"amount"`
}

// UniversalTransferResponse represents a universal transfer response.
type UniversalTransferResponse struct {
	TransactionID       string `json:"tranId"`
	ClientTransactionID string `json:"clientTranId"`
}

// UniversalTransfersDetail represents a list of universal transfers.
type UniversalTransfersDetail struct {
	Result []struct {
		TransactionID   int64        `json:"tranId"`
		FromEmail       string       `json:"fromEmail"`
		ToEmail         string       `json:"toEmail"`
		Asset           string       `json:"asset"`
		Amount          types.Number `json:"amount"`
		CreateTimeStamp types.Time   `json:"createTimeStamp"`
		FromAccountType string       `json:"fromAccountType"`
		ToAccountType   string       `json:"toAccountType"`
		Status          string       `json:"status"`
		ClientTranID    string       `json:"clientTranId"`
	} `json:"result"`
	TotalCount int64 `json:"totalCount"`
}

// MarginedFuturesAccount sub-account's futures account
type MarginedFuturesAccount struct {
	FutureAccountResp struct {
		Email  string `json:"email"`
		Assets []struct {
			Asset                  string       `json:"asset"`
			InitialMargin          types.Number `json:"initialMargin"`
			MaintenanceMargin      types.Number `json:"maintenanceMargin"`
			MarginBalance          types.Number `json:"marginBalance"`
			MaxWithdrawAmount      types.Number `json:"maxWithdrawAmount"`
			OpenOrderInitialMargin types.Number `json:"openOrderInitialMargin"`
			PositionInitialMargin  types.Number `json:"positionInitialMargin"`
			UnrealizedProfit       types.Number `json:"unrealizedProfit"`
			WalletBalance          types.Number `json:"walletBalance"`
		} `json:"assets"`
		CanDeposit                  bool         `json:"canDeposit"`
		CanTrade                    bool         `json:"canTrade"`
		CanWithdraw                 bool         `json:"canWithdraw"`
		FeeTier                     int64        `json:"feeTier"`
		MaxWithdrawAmount           types.Number `json:"maxWithdrawAmount"`
		TotalInitialMargin          types.Number `json:"totalInitialMargin"`
		TotalMaintenanceMargin      types.Number `json:"totalMaintenanceMargin"`
		TotalMarginBalance          types.Number `json:"totalMarginBalance"`
		TotalOpenOrderInitialMargin types.Number `json:"totalOpenOrderInitialMargin"`
		TotalPositionInitialMargin  types.Number `json:"totalPositionInitialMargin"`
		TotalUnrealizedProfit       types.Number `json:"totalUnrealizedProfit"`
		TotalWalletBalance          types.Number `json:"totalWalletBalance"`
		UpdateTime                  types.Time   `json:"updateTime"`
	} `json:"futureAccountResp"`
}

// AccountSummary represents sub-account's futures accounts for master account.
type AccountSummary struct {
	FutureAccountSummaryResp struct {
		TotalInitialMargin          types.Number `json:"totalInitialMargin"`
		TotalMaintenanceMargin      types.Number `json:"totalMaintenanceMargin"`
		TotalMarginBalance          types.Number `json:"totalMarginBalance"`
		TotalOpenOrderInitialMargin types.Number `json:"totalOpenOrderInitialMargin"`
		TotalPositionInitialMargin  types.Number `json:"totalPositionInitialMargin"`
		TotalUnrealizedProfit       types.Number `json:"totalUnrealizedProfit"`
		TotalWalletBalance          types.Number `json:"totalWalletBalance"`
		Asset                       string       `json:"asset"`
		SubAccountList              []struct {
			Email                       string       `json:"email"`
			TotalInitialMargin          types.Number `json:"totalInitialMargin"`
			TotalMaintenanceMargin      types.Number `json:"totalMaintenanceMargin"`
			TotalMarginBalance          types.Number `json:"totalMarginBalance"`
			TotalOpenOrderInitialMargin types.Number `json:"totalOpenOrderInitialMargin"`
			TotalPositionInitialMargin  types.Number `json:"totalPositionInitialMargin"`
			TotalUnrealizedProfit       types.Number `json:"totalUnrealizedProfit"`
			TotalWalletBalance          types.Number `json:"totalWalletBalance"`
			Asset                       string       `json:"asset"`
		} `json:"subAccountList"`
	} `json:"futureAccountSummaryResp,omitempty"`
	DeliveryAccountSummaryResp struct {
		TotalMarginBalanceOfBTC    types.Number `json:"totalMarginBalanceOfBTC"`
		TotalUnrealizedProfitOfBTC types.Number `json:"totalUnrealizedProfitOfBTC"`
		TotalWalletBalanceOfBTC    types.Number `json:"totalWalletBalanceOfBTC"`
		Asset                      string       `json:"asset"`
		SubAccountList             []struct {
			Email                 string       `json:"email"`
			TotalMarginBalance    types.Number `json:"totalMarginBalance"`
			TotalUnrealizedProfit types.Number `json:"totalUnrealizedProfit"`
			TotalWalletBalance    types.Number `json:"totalWalletBalance"`
			Asset                 string       `json:"asset"`
		} `json:"subAccountList"`
	} `json:"deliveryAccountSummaryResp,omitempty"`
}

// LeverageToken represents leveraged tokens for sub-accounts.
type LeverageToken struct {
	Email      string `json:"email"`
	EnableBlvt bool   `json:"enableBlvt"`
}

// APIRestrictions holds list of Ip addresses restricted to access the API key.
type APIRestrictions struct {
	IPRestrict string     `json:"ipRestrict"`
	IPList     []string   `json:"ipList"`
	UpdateTime types.Time `json:"updateTime"`
	APIKey     string     `json:"apiKey"`
}

// ManagedSubAccountAssetInfo represents managed sub-account asset information
type ManagedSubAccountAssetInfo struct {
	Coin             string       `json:"coin"`
	Name             string       `json:"name"`
	TotalBalance     types.Number `json:"totalBalance"`
	AvailableBalance types.Number `json:"availableBalance"`
	InOrder          string       `json:"inOrder"`
	BTCValue         types.Number `json:"btcValue"`
}

// SubAccountAssetsSnapshot represents a subaccount asset snapshot
type SubAccountAssetsSnapshot struct {
	Code        int64  `json:"code"`
	Msg         string `json:"msg"`
	SnapshotVos []struct {
		Data struct {
			Balances []struct {
				Asset  string       `json:"asset"`
				Free   types.Number `json:"free"`
				Locked types.Number `json:"locked"`
			} `json:"balances"`
			TotalAssetOfBTC types.Number `json:"totalAssetOfBtc"`
		} `json:"data"`
		Type       string     `json:"type"`
		UpdateTime types.Time `json:"updateTime"`
	} `json:"snapshotVos"`
}

// SubAccountTransferLog represents a managed sub-account transfer log
type SubAccountTransferLog struct {
	ManagerSubTransferHistoryVos []struct {
		FromEmail       string       `json:"fromEmail"`
		FromAccountType string       `json:"fromAccountType"`
		ToEmail         string       `json:"toEmail"`
		ToAccountType   string       `json:"toAccountType"`
		Asset           string       `json:"asset"`
		Amount          types.Number `json:"amount"`
		ScheduledData   types.Time   `json:"scheduledData"`
		CreateTime      types.Time   `json:"createTime"`
		Status          string       `json:"status"`
		TranID          int64        `json:"tranId"`
	} `json:"managerSubTransferHistoryVos"`
	Count int64 `json:"count"`
}

// ManagedSubAccountFuturesAssetDetail represents sub-accounts futures asset details.
type ManagedSubAccountFuturesAssetDetail struct {
	Code        string `json:"code"`
	Message     string `json:"message"`
	SnapshotVos []struct {
		Type       string     `json:"type"`
		UpdateTime types.Time `json:"updateTime"`
		Data       struct {
			Assets []struct {
				Asset         string       `json:"asset"`
				MarginBalance types.Number `json:"marginBalance"`
				WalletBalance types.Number `json:"walletBalance"`
			} `json:"assets"`
			Position []struct {
				Symbol      string       `json:"symbol"`
				EntryPrice  types.Number `json:"entryPrice"`
				MarkPrice   types.Number `json:"markPrice"`
				PositionAmt types.Number `json:"positionAmt"`
			} `json:"position"`
		} `json:"data"`
	} `json:"snapshotVos"`
}

// SubAccountMarginAsset represents a sub-account margin asset details response
type SubAccountMarginAsset struct {
	MarginLevel         string       `json:"marginLevel"`
	TotalAssetOfBTC     types.Number `json:"totalAssetOfBtc"`
	TotalLiabilityOfBTC types.Number `json:"totalLiabilityOfBtc"`
	TotalNetAssetOfBTC  types.Number `json:"totalNetAssetOfBtc"`
	UserAssets          []struct {
		Asset    string       `json:"asset"`
		Borrowed types.Number `json:"borrowed"`
		Free     types.Number `json:"free"`
		Interest types.Number `json:"interest"`
		Locked   types.Number `json:"locked"`
		NetAsset types.Number `json:"netAsset"`
	} `json:"userAssets"`
}

// ManagedSubAccountList represents a managed sub-account list.
type ManagedSubAccountList struct {
	Total                    int64 `json:"total"`
	ManagerSubUserInfoVoList []struct {
		RootUserID               int64      `json:"rootUserId"`
		ManagersubUserID         int64      `json:"managersubUserId"`
		BindParentUserID         int64      `json:"bindParentUserId"`
		Email                    string     `json:"email"`
		InsertTimeStamp          types.Time `json:"insertTimeStamp"`
		BindParentEmail          string     `json:"bindParentEmail"`
		IsSubUserEnabled         bool       `json:"isSubUserEnabled"`
		IsUserActive             bool       `json:"isUserActive"`
		IsMarginEnabled          bool       `json:"isMarginEnabled"`
		IsFutureEnabled          bool       `json:"isFutureEnabled"`
		IsSignedLVTRiskAgreement bool       `json:"isSignedLVTRiskAgreement"`
	} `json:"managerSubUserInfoVoList"`
}

// SubAccountTransactionStatistics holds a sub-account transaction statistics
type SubAccountTransactionStatistics struct {
	Recent30BtcTotal         types.Number `json:"recent30BtcTotal"`
	Recent30BtcFuturesTotal  types.Number `json:"recent30BtcFuturesTotal"`
	Recent30BtcMarginTotal   types.Number `json:"recent30BtcMarginTotal"`
	Recent30BusdTotal        types.Number `json:"recent30BusdTotal"`
	Recent30BusdFuturesTotal types.Number `json:"recent30BusdFuturesTotal"`
	Recent30BusdMarginTotal  types.Number `json:"recent30BusdMarginTotal"`
	TradeInfoVos             []struct {
		UserID      int64        `json:"userId"`
		BTC         types.Number `json:"btc"`
		BTCFutures  types.Number `json:"btcFutures"`
		BTCMargin   types.Number `json:"btcMargin"`
		BUSD        types.Number `json:"busd"`
		BUSDFutures types.Number `json:"busdFutures"`
		BUSDMargin  types.Number `json:"busdMargin"`
		Date        types.Time   `json:"date"`
	} `json:"tradeInfoVos"`
}

// ManagedSubAccountDepositAddres holds managed sub-account deposit address.
type ManagedSubAccountDepositAddres struct {
	Coin    string `json:"coin"`
	Address string `json:"address"`
	Tag     string `json:"tag"`
	URL     string `json:"url"`
}

// OptionsEnablingResponse holds a response after enabling options for sub-account
type OptionsEnablingResponse struct {
	Email             string `json:"email"`
	IsEOptionsEnabled bool   `json:"isEOptionsEnabled"`
}

// ManagedSubAccountTransferLog represents an asset transfer logs for trading team sub-accounts
type ManagedSubAccountTransferLog struct {
	ManagerSubTransferHistoryVos []struct {
		FromEmail       string       `json:"fromEmail"`
		FromAccountType string       `json:"fromAccountType"`
		ToEmail         string       `json:"toEmail"`
		ToAccountType   string       `json:"toAccountType"`
		Asset           string       `json:"asset"`
		Amount          types.Number `json:"amount"`
		CreateTime      types.Time   `json:"createTime"`
		ScheduledData   int64        `json:"scheduledData"`
		Status          string       `json:"status"`
		TranID          int64        `json:"tranId"`
	} `json:"managerSubTransferHistoryVos"`
	Count int64 `json:"count"`
}

// OCOOrderParam request parameter to place an OCO order.
type OCOOrderParam struct {
	Symbol                  currency.Pair `json:"symbol"`
	ListClientOrderID       string        `json:"listClientOrderId,omitempty"`
	Side                    string        `json:"side"`
	Amount                  float64       `json:"quantity"`
	LimitClientOrderID      string        `json:"limitClientOrderId,omitempty"`
	StopClientOrderID       string        `json:"stopClientOrderId,omitempty"`
	Price                   float64       `json:"price"`
	LimitIcebergQuantity    float64       `json:"limitIcebergQty,omitempty"`
	StopPrice               float64       `json:"stopPrice"`
	StopLimitPrice          float64       `json:"stopLimitPrice,omitempty"`
	StopIcebergQuantity     float64       `json:"stopIcebergQty,omitempty"`
	StopLimitTimeInForce    string        `json:"stopLimitTimeInForce,omitempty"` // Valid values are GTC/FOK/IOC
	NewOrderRespType        string        `json:"newOrderRespType,omitempty"`
	SideEffectType          string        `json:"sideEffectType"`                    // NO_SIDE_EFFECT, MARGIN_BUY, AUTO_REPAY; default NO_SIDE_EFFECT.
	SelfTradePreventionMode string        `json:"selfTradePreventionMode,omitempty"` // NONE:No STP / EXPIRE_TAKER:expire taker order when STP triggers/ EXPIRE_MAKER:expire taker order when STP triggers/ EXPIRE_BOTH:expire both orders when STP triggers

	// Only used with rest endpoint call
	TrailingDelta     int64  `json:"trailingDelta,omitempty"`
	LimitStrategyID   string `json:"limitStrategyId,omitempty"`
	LimitStrategyType string `json:"limitStrategyType,omitempty"`
	StopStrategyID    int64  `json:"stopStrategyId,omitempty"`
	StopStrategyType  int64  `json:"stopStrategyType,omitempty"`
}

// OCOOrderListParams represents an order parameter of OCO order as a list.
type OCOOrderListParams struct {
	Symbol                  string  `json:"symbol"`
	ListClientOrderID       string  `json:"listClientOrderId"`
	Side                    string  `json:"side"`
	Quantity                float64 `json:"quantity"`
	AboveType               string  `json:"aboveType"`
	AboveClientOrderID      string  `json:"aboveClientOrderId"`
	AboveIcebergQuantity    int64   `json:"aboveIcebergQty"`
	AbovePrice              float64 `json:"abovePrice"`
	AboveStopPrice          float64 `json:"aboveStopPrice"` // Can be used if aboveType is STOP_LOSS or STOP_LOSS_LIMIT. Either aboveStopPrice or aboveTrailingDelta or both, must be specified.
	AboveTrailingDelta      int64   `json:"aboveTrailingDelta"`
	AboveTimeInForce        string  `json:"aboveTimeInForce"` // Required if the aboveType is 'STOP_LOSS_LIMIT'.
	AboveStrategyID         int64   `json:"aboveStrategyId"`
	AboveStrategyType       int64   `json:"aboveStrategyType"`
	BelowType               string  `json:"belowType"` // Supported values : 'STOP_LOSS_LIMIT', 'STOP_LOSS', 'LIMIT_MAKER'
	BelowClientOrderID      string  `json:"belowClientOrderId"`
	BelowIcebergQty         int64   `json:"belowIcebergQty"` // Note that this can only be used if belowTimeInForce is 'GTC'.
	BelowPrice              float64 `json:"belowPrice"`
	BelowStopPrice          float64 `json:"belowStopPrice"` // Can be used if belowType is 'STOP_LOSS' or 'STOP_LOSS_LIMIT'. Either belowStopPrice or belowTrailingDelta or both, must be specified.
	BelowTrailingDelta      int64   `json:"belowTrailingDelta"`
	BelowTimeInForce        string  `json:"belowTimeInForce"` // Required if the belowType is 'STOP_LOSS_LIMIT'.
	BelowStrategyID         int64   `json:"belowStrategyId"`
	BelowStrategyType       int64   `json:"belowStrategyType"`
	NewOrderRespType        string  `json:"newOrderRespType"` // Select response format: 'ACK', 'RESULT', 'FULL'
	SelfTradePreventionMode string  `json:"selfTradePreventionMode"`
}

// OCOListOrderResponse represents a response for an OCO order list
type OCOListOrderResponse struct {
	OrderListID       int64      `json:"orderListId"`
	ContingencyType   string     `json:"contingencyType"`
	ListStatusType    string     `json:"listStatusType"`
	ListOrderStatus   string     `json:"listOrderStatus"`
	ListClientOrderID string     `json:"listClientOrderId"`
	TransactionTime   types.Time `json:"transactionTime"`
	Symbol            string     `json:"symbol"`
	Orders            []struct {
		Symbol        string `json:"symbol"`
		OrderID       int64  `json:"orderId"`
		ClientOrderID string `json:"clientOrderId"`
	} `json:"orders"`
	OrderReports []struct {
		Symbol                  string       `json:"symbol"`
		OrderID                 int64        `json:"orderId"`
		OrderListID             int64        `json:"orderListId"`
		ClientOrderID           string       `json:"clientOrderId"`
		TransactTime            types.Time   `json:"transactTime"`
		Price                   types.Number `json:"price"`
		OrigQty                 types.Number `json:"origQty"`
		ExecutedQty             types.Number `json:"executedQty"`
		CummulativeQuoteQty     types.Number `json:"cummulativeQuoteQty"`
		Status                  string       `json:"status"`
		TimeInForce             string       `json:"timeInForce"`
		Type                    string       `json:"type"`
		Side                    string       `json:"side"`
		StopPrice               types.Number `json:"stopPrice,omitempty"`
		WorkingTime             types.Time   `json:"workingTime"`
		IcebergQty              types.Number `json:"icebergQty,omitempty"`
		SelfTradePreventionMode string       `json:"selfTradePreventionMode"`
	} `json:"orderReports"`
}

// SOROrderRequestParams represents a request parameters for SOR orders.
type SOROrderRequestParams struct {
	Symbol                  currency.Pair `json:"symbol"`
	Side                    string        `json:"side,omitempty"`
	OrderType               string        `json:"type,omitempty"`
	TimeInForce             string        `json:"timeInForce,omitempty"`
	Quantity                float64       `json:"quantity"`
	Price                   float64       `json:"price"`
	NewClientOrderID        string        `json:"newClientOrderId,omitempty"`
	StrategyID              int64         `json:"strategyId,omitempty"`
	StrategyType            int64         `json:"strategyType,omitempty"`
	IcebergQuantity         float64       `json:"icebergQty,omitempty"`              // Used with 'LIMIT' to create an iceberg order.
	NewOrderResponseType    string        `json:"newOrderRespType,omitempty"`        // Set the response JSON. 'ACK', 'RESULT', or 'FULL'. Default to 'FULL'
	SelfTradePreventionMode string        `json:"selfTradePreventionMode,omitempty"` // The allowed enums is dependent on what is configured on the symbol. The possible supported values are 'EXPIRE_TAKER', 'EXPIRE_MAKER', 'EXPIRE_BOTH', 'NONE'.
}

// SOROrderResponse represents smart order routing response instance.
type SOROrderResponse struct {
	Symbol              string       `json:"symbol"`
	OrderID             int64        `json:"orderId"`
	OrderListID         int64        `json:"orderListId"`
	ClientOrderID       string       `json:"clientOrderId"`
	TransactTime        types.Time   `json:"transactTime"`
	Price               types.Number `json:"price"`
	OrigQty             types.Number `json:"origQty"`
	ExecutedQty         types.Number `json:"executedQty"`
	CummulativeQuoteQty types.Number `json:"cummulativeQuoteQty"`
	Status              string       `json:"status"`
	TimeInForce         string       `json:"timeInForce"`
	Type                string       `json:"type"`
	Side                string       `json:"side"`
	WorkingTime         types.Time   `json:"workingTime"`
	Fills               []struct {
		MatchType       string       `json:"matchType"`
		Price           types.Number `json:"price"`
		Qty             types.Number `json:"qty"`
		Commission      string       `json:"commission"`
		CommissionAsset string       `json:"commissionAsset"`
		TradeID         int64        `json:"tradeId"`
		AllocID         int64        `json:"allocId"`
	} `json:"fills"`
	WorkingFloor            string `json:"workingFloor"`
	SelfTradePreventionMode string `json:"selfTradePreventionMode"`
	UsedSor                 bool   `json:"usedSor"`
}

// AccountTradeItem represents an account trade item detail.
type AccountTradeItem struct {
	Symbol          string       `json:"symbol"`
	ID              int64        `json:"id"`
	OrderID         int64        `json:"orderId"`
	OrderListID     int64        `json:"orderListId"`
	Price           types.Number `json:"price"`
	Qty             types.Number `json:"qty"`
	QuoteQty        types.Number `json:"quoteQty"`
	Commission      string       `json:"commission"`
	CommissionAsset string       `json:"commissionAsset"`
	Time            types.Time   `json:"time"`
	IsBuyer         bool         `json:"isBuyer"`
	IsMaker         bool         `json:"isMaker"`
	IsBestMatch     bool         `json:"isBestMatch"`
}

// CurrentOrderCountUsage user's current order count usage for all intervals.
type CurrentOrderCountUsage struct {
	RateLimitType string `json:"rateLimitType"`
	Interval      string `json:"interval"`
	IntervalNum   int64  `json:"intervalNum"`
	Limit         int64  `json:"limit"`
	Count         int64  `json:"count"`
}

// PreventedMatches represents user's prevented matches
type PreventedMatches struct {
	Symbol                  string       `json:"symbol"`
	PreventedMatchID        int64        `json:"preventedMatchId"`
	TakerOrderID            int64        `json:"takerOrderId"`
	MakerOrderID            int64        `json:"makerOrderId"`
	TradeGroupID            int64        `json:"tradeGroupId"`
	SelfTradePreventionMode string       `json:"selfTradePreventionMode"`
	Price                   types.Number `json:"price"`
	MakerPreventedQuantity  types.Number `json:"makerPreventedQuantity"`
	TransactTime            types.Time   `json:"transactTime"`
}

// Allocation represents Smart Order Routing (SOR) order allocation
type Allocation struct {
	Symbol          string       `json:"symbol"`
	AllocationID    int64        `json:"allocationId"`
	AllocationType  string       `json:"allocationType"`
	OrderID         int64        `json:"orderId"`
	OrderListID     int64        `json:"orderListId"`
	Price           types.Number `json:"price"`
	Quantity        types.Number `json:"qty"`
	QuoteQty        types.Number `json:"quoteQty"`
	Commission      string       `json:"commission"`
	CommissionAsset string       `json:"commissionAsset"`
	Time            types.Time   `json:"time"`
	IsBuyer         bool         `json:"isBuyer"`
	IsMaker         bool         `json:"isMaker"`
	IsAllocator     bool         `json:"isAllocator"`
}

// AccountCommissionRate represents an account commission rate
type AccountCommissionRate struct {
	Symbol             string `json:"symbol"`
	StandardCommission struct {
		Maker  types.Number `json:"maker"`
		Taker  types.Number `json:"taker"`
		Buyer  types.Number `json:"buyer"`
		Seller types.Number `json:"seller"`
	} `json:"standardCommission"`
	TaxCommission struct {
		Maker  types.Number `json:"maker"`
		Taker  types.Number `json:"taker"`
		Buyer  types.Number `json:"buyer"`
		Seller types.Number `json:"seller"`
	} `json:"taxCommission"`
	Discount struct {
		EnabledForAccount bool         `json:"enabledForAccount"`
		EnabledForSymbol  bool         `json:"enabledForSymbol"`
		DiscountAsset     string       `json:"discountAsset"`
		Discount          types.Number `json:"discount"`
	} `json:"discount"`
}

// MarginAccountBorrowRepayRecords represents borrow/repay records in Margin account.
type MarginAccountBorrowRepayRecords struct {
	Rows []struct {
		IsolatedSymbol string       `json:"isolatedSymbol"`
		Amount         types.Number `json:"amount"`
		Asset          string       `json:"asset"`
		Interest       string       `json:"interest"`
		Principal      string       `json:"principal"`
		Status         string       `json:"status"`
		Timestamp      types.Time   `json:"timestamp"`
		TransactionID  int64        `json:"txId"`
	} `json:"rows"`
	Total int64 `json:"total"`
}

// MarginAsset represents margin asset instance.
type MarginAsset struct {
	AssetFullName  string       `json:"assetFullName"`
	AssetName      string       `json:"assetName"`
	IsBorrowable   bool         `json:"isBorrowable"`
	IsMortgageable bool         `json:"isMortgageable"`
	UserMinBorrow  types.Number `json:"userMinBorrow"`
	UserMinRepay   types.Number `json:"userMinRepay"`
	DelistTime     types.Time   `json:"delistTime"`
}

// CrossMarginPairInfo holds cross margin symbols and detail.
type CrossMarginPairInfo struct {
	Base          string     `json:"base"`
	ID            int64      `json:"id"`
	IsBuyAllowed  bool       `json:"isBuyAllowed"`
	IsMarginTrade bool       `json:"isMarginTrade"`
	IsSellAllowed bool       `json:"isSellAllowed"`
	Quote         string     `json:"quote"`
	Symbol        string     `json:"symbol"`
	DelistTime    types.Time `json:"delistTime,omitempty"`
}

// MarginPriceIndex represents margin account price index
type MarginPriceIndex struct {
	CalcTime types.Time   `json:"calcTime"`
	Price    types.Number `json:"price"`
	Symbol   string       `json:"symbol"`
}

// MarginAccountOrderParam represents a margin account order.
type MarginAccountOrderParam struct {
	Symbol     currency.Pair `json:"symbol"`
	IsIsolated bool          `json:"isIsolated,string"`
	Side       string        `json:"side"`
	OrderType  string        `json:"type"`

	Quantity                float64 `json:"quantity,omitempty"`
	QuoteOrderQuantity      float64 `json:"quoteOrderQty,omitempty"`
	Price                   float64 `json:"price,omitempty"`
	StopPrice               float64 `json:"stopPrice,omitempty"` // Used with STOP_LOSS, STOP_LOSS_LIMIT, TAKE_PROFIT, and TAKE_PROFIT_LIMIT orders.
	NewClientOrderID        string  `json:"newClientOrderId,omitempty"`
	IcebergQuantity         float64 `json:"icebergQty,omitempty"` // Used with LIMIT, STOP_LOSS_LIMIT, and TAKE_PROFIT_LIMIT to create an iceberg order.
	NewOrderResponseType    string  `json:"newOrderRespType,omitempty"`
	SideEffectType          string  `json:"sideEffectType,omitempty"` // NO_SIDE_EFFECT, MARGIN_BUY, AUTO_REPAY,AUTO_BORROW_REPAY; default NO_SIDE_EFFECT.
	TimeInForce             string  `json:"timeInForce,omitempty"`
	SelfTradePreventionMode string  `json:"selfTradePreventionMode,omitempty"`
	AutoRepayAtCancel       bool    `json:"autoRepayAtCancel,omitempty"`
}

// MarginAccountOrder represents a margin account order.
type MarginAccountOrder struct {
	Symbol                  string       `json:"symbol"`
	OrderID                 int64        `json:"orderId"`
	ClientOrderID           string       `json:"clientOrderId"`
	OrigClientOrderID       string       `json:"origClientOrderId"`
	TransactTime            types.Time   `json:"transactTime"`
	Price                   types.Number `json:"price"`
	OrigQty                 types.Number `json:"origQty"`
	ExecutedQty             types.Number `json:"executedQty"`
	CummulativeQuoteQty     string       `json:"cummulativeQuoteQty"`
	Status                  string       `json:"status"`
	TimeInForce             string       `json:"timeInForce"`
	OrderType               string       `json:"type"`
	IsIsolated              bool         `json:"isIsolated"`
	Side                    string       `json:"side"`
	SelfTradePreventionMode string       `json:"selfTradePreventionMode"`
}

// MarginAccountOrderDetail represents orders on symbol for margin account.
type MarginAccountOrderDetail struct {
	MarginAccountOrder

	ContingencyType   string     `json:"contingencyType,omitempty"`
	ListStatusType    string     `json:"listStatusType,omitempty"`
	ListOrderStatus   string     `json:"listOrderStatus,omitempty"`
	ListClientOrderID string     `json:"listClientOrderId,omitempty"`
	TransactionTime   types.Time `json:"transactionTime,omitempty"`
	Orders            []struct {
		Symbol        string `json:"symbol"`
		OrderID       int    `json:"orderId"`
		ClientOrderID string `json:"clientOrderId"`
	} `json:"orders,omitempty"`
	OrderReports []struct {
		Symbol              string       `json:"symbol"`
		OrigClientOrderID   string       `json:"origClientOrderId"`
		OrderID             int64        `json:"orderId"`
		OrderListID         int64        `json:"orderListId"`
		ClientOrderID       string       `json:"clientOrderId"`
		Price               types.Number `json:"price"`
		OrigQty             types.Number `json:"origQty"`
		ExecutedQty         types.Number `json:"executedQty"`
		CummulativeQuoteQty types.Number `json:"cummulativeQuoteQty"`
		Status              string       `json:"status"`
		TimeInForce         string       `json:"timeInForce"`
		OrderType           string       `json:"type"`
		Side                string       `json:"side"`
		StopPrice           string       `json:"stopPrice,omitempty"`
		IcebergQty          string       `json:"icebergQty"`
	} `json:"orderReports,omitempty"`
}

// CrossMarginTransferHistory represents a cross-margin transfer history
type CrossMarginTransferHistory struct {
	Rows []struct {
		Amount        types.Number `json:"amount"`
		Asset         string       `json:"asset"`
		Status        string       `json:"status"`
		Timestamp     types.Time   `json:"timestamp"`
		TransactionID int64        `json:"txId"`
		TransferType  string       `json:"type"`
		TransferFrom  string       `json:"transFrom,omitempty"`
		TransferTo    string       `json:"transTo,omitempty"` // SPOT,FUTURES,FIAT,DELIVERY,MINING,ISOLATED_MARGIN,FUNDING,MOTHER_SPOT,OPTION,SUB_SPOT,SUB_MARGIN,CROSS_MARGIN
		FromSymbol    string       `json:"fromSymbol,omitempty"`
		ToSymbol      string       `json:"toSymbol,omitempty"`
	} `json:"rows"`
	Total int64 `json:"total"`
}

// LiquidiationRecord represents a liquidiation record history
type LiquidiationRecord struct {
	Rows []struct {
		AvgPrice    types.Number `json:"avgPrice"`
		ExecutedQty types.Number `json:"executedQty"`
		OrderID     int64        `json:"orderId"`
		Price       types.Number `json:"price"`
		Qty         types.Number `json:"qty"`
		Side        string       `json:"side"`
		Symbol      string       `json:"symbol"`
		TimeInForce string       `json:"timeInForce"`
		IsIsolated  bool         `json:"isIsolated"`
		UpdatedTime types.Time   `json:"updatedTime"`
	} `json:"rows"`
	Total int64 `json:"total"`
}

// CrossMarginAccount represents cross margin account detail
type CrossMarginAccount struct {
	BorrowEnabled              bool         `json:"borrowEnabled"`
	MarginLevel                types.Number `json:"marginLevel"`
	CollateralMarginLevel      types.Number `json:"CollateralMarginLevel"`
	TotalAssetOfBTC            types.Number `json:"totalAssetOfBtc"`
	TotalNetAssetOfBTC         types.Number `json:"totalNetAssetOfBtc"`
	TotalLiabilityOfBTC        types.Number `json:"totalLiabilityOfBtc"`
	TotalCollateralValueInUSDT types.Number `json:"TotalCollateralValueInUSDT"`
	TradeEnabled               bool         `json:"tradeEnabled"`
	TransferEnabled            bool         `json:"transferEnabled"`
	AccountType                string       `json:"accountType"`
	UserAssets                 []struct {
		Asset    string       `json:"asset"`
		Borrowed types.Number `json:"borrowed"`
		Free     types.Number `json:"free"`
		Interest types.Number `json:"interest"`
		Locked   types.Number `json:"locked"`
		NetAsset types.Number `json:"netAsset"`
	} `json:"userAssets"`
}

// MarginOCOOrderParam represents an OCO order parameters.
type MarginOCOOrderParam struct {
	Symbol                  currency.Pair `json:"symbol"`
	IsIsolated              bool          `json:"isIsolated,omitempty,string"`
	ListClientOrderID       string        `json:"listClientOrderId,omitempty"`
	Side                    string        `json:"side"`
	Quantity                float64       `json:"quantity"`
	LimitClientOrderID      string        `json:"limitClientOrderId,omitempty"`
	Price                   float64       `json:"price"`
	LimitIcebergQuantity    float64       `json:"limitIcebergQty,omitempty"`
	StopClientOrderID       string        `json:"stopClientOrderId"`
	StopPrice               float64       `json:"stopPrice,omitempty"`
	StopLimitPrice          float64       `json:"stopLimitPrice,omitempty"` // If provided, stopLimitTimeInForce is required.
	StopIcebergQuantity     float64       `json:"stopIcebergQty,omitempty"`
	StopLimitTimeInForce    float64       `json:"stopLimitTimeInForce,omitempty"` // Valid values are GTC/FOK/IOC
	NewOrderRespType        string        `json:"newOrderRespType,omitempty"`
	SideEffectType          string        `json:"sideEffectType,omitempty"` // NO_SIDE_EFFECT, MARGIN_BUY, AUTO_REPAY,AUTO_BORROW_REPAY; default NO_SIDE_EFFECT.
	SelfTradePreventionMode string        `json:"selfTradePreventionMode,omitempty"`
	AutoRepayAtCancel       string        `json:"autoRepayAtCancel,omitempty"`
}

// MarginAccountSummary represents a margin account summary information.
type MarginAccountSummary struct {
	NormalBar           types.Number `json:"normalBar"`
	MarginCallBar       types.Number `json:"marginCallBar"`
	ForceLiquidationBar types.Number `json:"forceLiquidationBar"`
}

// IsolatedMarginAccountInfo represents isolated margin account detail.
type IsolatedMarginAccountInfo struct {
	Assets []struct {
		BaseAsset         AssetInfo    `json:"baseAsset"`
		QuoteAsset        AssetInfo    `json:"quoteAsset"`
		Symbol            string       `json:"symbol"`
		IsolatedCreated   bool         `json:"isolatedCreated"`
		Enabled           bool         `json:"enabled"`
		MarginLevel       string       `json:"marginLevel"`
		MarginLevelStatus string       `json:"marginLevelStatus"`
		MarginRatio       types.Number `json:"marginRatio"`
		IndexPrice        types.Number `json:"indexPrice"`
		LiquidatePrice    types.Number `json:"liquidatePrice"`
		LiquidateRate     types.Number `json:"liquidateRate"`
		TradeEnabled      bool         `json:"tradeEnabled"`
	} `json:"assets"`
	TotalAssetOfBTC     types.Number `json:"totalAssetOfBtc"`
	TotalLiabilityOfBTC types.Number `json:"totalLiabilityOfBtc"`
	TotalNetAssetOfBTC  types.Number `json:"totalNetAssetOfBtc"`
}

// AssetInfo represents an asset isolated margin asset detail information
type AssetInfo struct {
	Asset         string       `json:"asset"`
	BorrowEnabled bool         `json:"borrowEnabled"`
	Borrowed      types.Number `json:"borrowed"`
	Free          types.Number `json:"free"`
	Interest      types.Number `json:"interest"`
	Locked        types.Number `json:"locked"`
	NetAsset      types.Number `json:"netAsset"`
	NetAssetOfBTC types.Number `json:"netAssetOfBtc"`
	RepayEnabled  bool         `json:"repayEnabled"`
	TotalAsset    types.Number `json:"totalAsset"`
}

// IsolatedMarginResponse represents an isolated margin account disable operation response.
type IsolatedMarginResponse struct {
	Success bool   `json:"success"`
	Symbol  string `json:"symbol"`
}

// IsolatedMarginAccountLimit represents isolated margin account limit info
type IsolatedMarginAccountLimit struct {
	EnabledAccount float64 `json:"enabledAccount"`
	MaxAccount     float64 `json:"maxAccount"`
}

// IsolatedMarginAccount represents an isolated margin account
type IsolatedMarginAccount struct {
	Base          string     `json:"base"`
	IsBuyAllowed  bool       `json:"isBuyAllowed"`
	IsMarginTrade bool       `json:"isMarginTrade"`
	IsSellAllowed bool       `json:"isSellAllowed"`
	Quote         string     `json:"quote"`
	Symbol        string     `json:"symbol"`
	DelistTime    types.Time `json:"delistTime,omitempty"`
}

// BNBBurnOnSpotAndMarginInterest represents a response of spot trade and margin interest
type BNBBurnOnSpotAndMarginInterest struct {
	SpotBNBBurn     bool `json:"spotBNBBurn"`
	InterestBNBBurn bool `json:"interestBNBBurn"`
}

// MarginInterestRate represents a margin interest rate item.
type MarginInterestRate struct {
	Asset             string     `json:"asset"`
	DailyInterestRate string     `json:"dailyInterestRate"`
	Timestamp         types.Time `json:"timestamp"`
	VipLevel          int64      `json:"vipLevel"`
}

// CrossMarginFeeData represents a cross margin fee detail
type CrossMarginFeeData struct {
	VipLevel        int64        `json:"vipLevel"`
	Coin            string       `json:"coin"`
	TransferIn      bool         `json:"transferIn"`
	Borrowable      bool         `json:"borrowable"`
	DailyInterest   types.Number `json:"dailyInterest"`
	YearlyInterest  types.Number `json:"yearlyInterest"`
	BorrowLimit     types.Number `json:"borrowLimit"`
	MarginablePairs []string     `json:"marginablePairs"`
}

// IsolatedMarginFeeData represents an isolated margin fee data.
type IsolatedMarginFeeData struct {
	VipLevel int64  `json:"vipLevel"`
	Symbol   string `json:"symbol"`
	Leverage string `json:"leverage"`
	Data     []struct {
		Coin          string       `json:"coin"`
		DailyInterest string       `json:"dailyInterest"`
		BorrowLimit   types.Number `json:"borrowLimit"`
	} `json:"data"`
}

// IsolatedMarginTierInfo represents isolated margin tier item.
type IsolatedMarginTierInfo struct {
	Symbol                  string       `json:"symbol"`
	Tier                    int64        `json:"tier"`
	EffectiveMultiple       types.Number `json:"effectiveMultiple"`
	InitialRiskRatio        types.Number `json:"initialRiskRatio"`
	LiquidationRiskRatio    types.Number `json:"liquidationRiskRatio"`
	BaseAssetMaxBorrowable  types.Number `json:"baseAssetMaxBorrowable"`
	QuoteAssetMaxBorrowable types.Number `json:"quoteAssetMaxBorrowable"`
}

// MarginOrderCount represents margin order count usage for an interval,
type MarginOrderCount struct {
	RateLimitType string `json:"rateLimitType"`
	Interval      string `json:"interval"`
	IntervalNum   int64  `json:"intervalNum"`
	Limit         int64  `json:"limit"`
	Count         int64  `json:"count"`
}

// CrossMarginCollateralRatio represents a cross-margin collateral ratio
type CrossMarginCollateralRatio struct {
	Collaterals []struct {
		MinUsdValue  types.Number `json:"minUsdValue"`
		MaxUsdValue  types.Number `json:"maxUsdValue,omitempty"`
		DiscountRate types.Number `json:"discountRate"`
	} `json:"collaterals"`
	AssetNames []string `json:"assetNames"`
}

// SmallLiabilityCoin represents information of coins which can be small liability
type SmallLiabilityCoin struct {
	Asset          string  `json:"asset"`
	Interest       string  `json:"interest"`
	Principal      string  `json:"principal"`
	LiabilityAsset string  `json:"liabilityAsset"`
	LiabilityQty   float64 `json:"liabilityQty"`
}

// SmallLiabilityExchange represents small liability exchange history.
type SmallLiabilityExchange struct {
	Total int `json:"total"`
	Rows  []struct {
		Asset        string       `json:"asset"`
		Amount       string       `json:"amount"`
		TargetAsset  string       `json:"targetAsset"`
		TargetAmount types.Number `json:"targetAmount"`
		BizType      string       `json:"bizType"`
		Timestamp    types.Time   `json:"timestamp"`
	} `json:"rows"`
}

// HourlyInterestrate represents an asset and it's hourlt interest rate information.
type HourlyInterestrate struct {
	Asset                  string       `json:"asset"`
	NextHourlyInterestRate types.Number `json:"nextHourlyInterestRate"`
}

// MarginCapitalFlow represents a cross-margin or isolated margin capital flow for an asset
type MarginCapitalFlow struct {
	ID            int64        `json:"id"`
	TransactionID int64        `json:"tranId"`
	Timestamp     types.Time   `json:"timestamp"`
	Asset         string       `json:"asset"`
	Symbol        string       `json:"symbol"`
	Type          string       `json:"type"`
	Amount        types.Number `json:"amount"`
}

// MarginDelistSchedule represents delist schedule for cross-margin and isolated-margin accounts.
type MarginDelistSchedule struct {
	DelistTime            types.Time `json:"delistTime"`
	CrossMarginAssets     []string   `json:"crossMarginAssets"`
	IsolatedMarginSymbols []string   `json:"isolatedMarginSymbols"`
}

// MarginInventory represents margin available inventory for each asset
type MarginInventory struct {
	Assets     map[string]types.Number `json:"assets"`
	UpdateTime types.Time              `json:"updateTime"`
}

// LiabilityCoinLeverageBracket represents liability coin leverage bracket in cross margin pro-mode
type LiabilityCoinLeverageBracket struct {
	AssetNames []string `json:"assetNames"`
	Rank       int64    `json:"rank"`
	Brackets   []struct {
		Leverage              int64        `json:"leverage"`
		MaxDebt               types.Number `json:"maxDebt"`
		MaintenanceMarginRate types.Number `json:"maintenanceMarginRate"`
		InitialMarginRate     types.Number `json:"initialMarginRate"`
		FastNum               types.Number `json:"fastNum"`
	} `json:"brackets"`
}

// SimpleEarnProducts represents list of binance's simple earn product
type SimpleEarnProducts struct {
	Rows []struct {
		Asset                      string             `json:"asset"`
		LatestAnnualPercentageRate types.Number       `json:"latestAnnualPercentageRate"`
		TierAnnualPercentageRate   map[string]float64 `json:"tierAnnualPercentageRate"`
		AirDropPercentageRate      string             `json:"airDropPercentageRate"`
		CanPurchase                bool               `json:"canPurchase"`
		CanRedeem                  bool               `json:"canRedeem"`
		IsSoldOut                  bool               `json:"isSoldOut"`
		Hot                        bool               `json:"hot"`
		MinPurchaseAmount          types.Number       `json:"minPurchaseAmount"`
		ProductID                  string             `json:"productId"`
		SubscriptionStartTime      types.Time         `json:"subscriptionStartTime"`
		Status                     string             `json:"status"`
	} `json:"rows"`
	Total int64 `json:"total"`
}

// LockedSimpleEarnProducts represents locked simple earn products.
type LockedSimpleEarnProducts struct {
	Rows []struct {
		ProjectID string `json:"projectId"`
		Detail    struct {
			Asset                 string     `json:"asset"`
			RewardAsset           string     `json:"rewardAsset"`
			Duration              int64      `json:"duration"`
			Renewable             bool       `json:"renewable"`
			IsSoldOut             bool       `json:"isSoldOut"`
			Apr                   string     `json:"apr"`
			Status                string     `json:"status"`
			SubscriptionStartTime types.Time `json:"subscriptionStartTime"`
			ExtraRewardAsset      string     `json:"extraRewardAsset"`
			ExtraRewardAPR        string     `json:"extraRewardAPR"`
		} `json:"detail"`
		Quota struct {
			TotalPersonalQuota types.Number `json:"totalPersonalQuota"`
			Minimum            types.Number `json:"minimum"`
		} `json:"quota"`
	} `json:"rows"`
	Total int64 `json:"total"`
}

// SimpleEarnSubscriptionResponse represents a simple earn product subscription response.
type SimpleEarnSubscriptionResponse struct {
	PurchaseID int64 `json:"purchaseId"`
	Success    bool  `json:"success"`

	PositionID string `json:"positionId"` // Sent when subscribing to locked simple earn products.
}

// RedeemResponse represents a simple flexible or locked product redemption response.
type RedeemResponse struct {
	RedeemID int64 `json:"redeemId"`
	Success  bool  `json:"success"`
}

// FlexibleProductPosition represents a flexible product position instance.
type FlexibleProductPosition struct {
	Rows []struct {
		TotalAmount                    types.Number       `json:"totalAmount"`
		TierAnnualPercentageRate       map[string]float64 `json:"tierAnnualPercentageRate"`
		LatestAnnualPercentageRate     types.Number       `json:"latestAnnualPercentageRate"`
		YesterdayAirdropPercentageRate types.Number       `json:"yesterdayAirdropPercentageRate"`
		Asset                          string             `json:"asset"`
		AirDropAsset                   string             `json:"airDropAsset"`
		CanRedeem                      bool               `json:"canRedeem"`
		CollateralAmount               types.Number       `json:"collateralAmount"`
		ProductID                      string             `json:"productId"`
		YesterdayRealTimeRewards       string             `json:"yesterdayRealTimeRewards"`
		CumulativeBonusRewards         string             `json:"cumulativeBonusRewards"`
		CumulativeRealTimeRewards      string             `json:"cumulativeRealTimeRewards"`
		CumulativeTotalRewards         string             `json:"cumulativeTotalRewards"`
		AutoSubscribe                  bool               `json:"autoSubscribe"`
	} `json:"rows"`
	Total int64 `json:"total"`
}

// LockedProductPosition represents locked product position instance.
type LockedProductPosition struct {
	Rows []struct {
		PositionID   string       `json:"positionId"`
		ProjectID    string       `json:"projectId"`
		Asset        string       `json:"asset"`
		Amount       types.Number `json:"amount"`
		PurchaseTime types.Time   `json:"purchaseTime"`
		Duration     string       `json:"duration"`
		AccrualDays  string       `json:"accrualDays"`
		RewardAsset  string       `json:"rewardAsset"`
		Apy          string       `json:"APY"`
		IsRenewable  bool         `json:"isRenewable"`
		IsAutoRenew  bool         `json:"isAutoRenew"`
		RedeemDate   string       `json:"redeemDate"`
	} `json:"rows"`
	Total int64 `json:"total"`
}

// SimpleAccount represents a simple account instance.
type SimpleAccount struct {
	TotalAmountInBTC          types.Number `json:"totalAmountInBTC"`
	TotalAmountInUSDT         types.Number `json:"totalAmountInUSDT"`
	TotalFlexibleAmountInBTC  types.Number `json:"totalFlexibleAmountInBTC"`
	TotalFlexibleAmountInUSDT types.Number `json:"totalFlexibleAmountInUSDT"`
	TotalLockedInBTC          types.Number `json:"totalLockedInBTC"`
	TotalLockedInUSDT         types.Number `json:"totalLockedInUSDT"`
}

// FlexibleSubscriptionRecord represents list of flexible subscriptions.
type FlexibleSubscriptionRecord struct {
	Rows []struct {
		Amount         types.Number `json:"amount"`
		Asset          string       `json:"asset"`
		Time           types.Time   `json:"time"`
		PurchaseID     int64        `json:"purchaseId"`
		Type           string       `json:"type"`
		SourceAccount  string       `json:"sourceAccount"`
		AmtFromSpot    string       `json:"amtFromSpot"`
		AmtFromFunding string       `json:"amtFromFunding"`
		Status         string       `json:"status"`
	} `json:"rows"`
	Total int64 `json:"total"`
}

// LockedSubscriptions represents locked subscription records instance.
type LockedSubscriptions struct {
	Rows []struct {
		PositionID     string       `json:"positionId"`
		PurchaseID     int64        `json:"purchaseId"`
		Time           types.Time   `json:"time"`
		Asset          string       `json:"asset"`
		Amount         types.Number `json:"amount"`
		LockPeriod     string       `json:"lockPeriod"`
		Type           string       `json:"type"`           // NORMAL for normal subscription, AUTO for auto-subscription order, ACTIVITY for activity order, TRIAL for trial fund order, RESTAKE for restake order
		SourceAccount  string       `json:"sourceAccount"`  // SPOT, FUNDING, SPOTANDFUNDING
		AmtFromSpot    string       `json:"amtFromSpot"`    // Display if sourceAccount is SPOTANDFUNDING
		AmtFromFunding string       `json:"amtFromFunding"` // Display if sourceAccount is SPOTANDFUNDING
		Status         string       `json:"status"`         // PURCHASING/SUCCESS/FAILED
	} `json:"rows"`
	Total int64 `json:"total"`
}

// RedemptionRecord represents a redemption instance.
type RedemptionRecord struct {
	Rows []struct {
		Amount      types.Number `json:"amount"`
		Asset       string       `json:"asset"`
		Time        types.Time   `json:"time"`
		ProjectID   string       `json:"projectId"`
		RedeemID    int          `json:"redeemId"`
		DestAccount string       `json:"destAccount"`
		Status      string       `json:"status"`
	} `json:"rows"`
	Total int64 `json:"total"`
}

// LockedRedemptionRecord represents a locked redemption record.
type LockedRedemptionRecord struct {
	Rows []struct {
		PositionID  string       `json:"positionId"`
		RedeemID    int64        `json:"redeemId"`
		Time        types.Time   `json:"time"`
		Asset       string       `json:"asset"`
		LockPeriod  string       `json:"lockPeriod"`
		Amount      types.Number `json:"amount"`
		Type        string       `json:"type"`
		DeliverDate string       `json:"deliverDate"`
		Status      string       `json:"status"`
	} `json:"rows"`
	Total int64 `json:"total"`
}

// FlexibleReward represents a flexible reward history item.
type FlexibleReward struct {
	Rows []struct {
		Asset     string     `json:"asset"`
		Rewards   string     `json:"rewards"`
		ProjectID string     `json:"projectId"`
		Type      string     `json:"type"`
		Time      types.Time `json:"time"`
	} `json:"rows"`
	Total int64 `json:"total"`
}

// LockedRewards represents locked rewards list.
type LockedRewards struct {
	Rows []struct {
		PositionID string     `json:"positionId"`
		Time       types.Time `json:"time"`
		Asset      string     `json:"asset"`
		LockPeriod string     `json:"lockPeriod"`
		Amount     string     `json:"amount"`
	} `json:"rows"`
	Total int64 `json:"total"`
}

// PersonalLeftQuota represents personal quota
type PersonalLeftQuota struct {
	LeftPersonalQuota string `json:"leftPersonalQuota"`
}

// FlexibleSubscriptionPreview represents a subscription preview for flexible assets.
type FlexibleSubscriptionPreview struct {
	TotalAmount             types.Number `json:"totalAmount"`
	RewardAsset             string       `json:"rewardAsset"`
	AirDropAsset            string       `json:"airDropAsset"`
	EstDailyBonusRewards    string       `json:"estDailyBonusRewards"`
	EstDailyRealTimeRewards string       `json:"estDailyRealTimeRewards"`
	EstDailyAirdropRewards  string       `json:"estDailyAirdropRewards"`
}

// LockedSubscriptionPreview represents a subscription preview for locked assets.
type LockedSubscriptionPreview struct {
	RewardAsset            string       `json:"rewardAsset"`
	TotalRewardAmt         types.Number `json:"totalRewardAmt"`
	ExtraRewardAsset       string       `json:"extraRewardAsset"`
	EstTotalExtraRewardAmt types.Number `json:"estTotalExtraRewardAmt"`
	NextPay                types.Number `json:"nextPay"`
	NextPayDate            string       `json:"nextPayDate"`
	ValueDate              string       `json:"valueDate"`
	RewardsEndDate         string       `json:"rewardsEndDate"`
	DeliverDate            string       `json:"deliverDate"`
	NextSubscriptionDate   string       `json:"nextSubscriptionDate"`
}

// SimpleEarnRateHistory represents a simple-earn rate history
type SimpleEarnRateHistory struct {
	Rows []struct {
		ProductID            string       `json:"productId"`
		Asset                string       `json:"asset"`
		AnnualPercentageRate types.Number `json:"annualPercentageRate"`
		Time                 types.Time   `json:"time"`
	} `json:"rows"`
	Total string `json:"total"`
}

// SimpleEarnCollateralRecords represents a collateral records of simple-earn products
type SimpleEarnCollateralRecords struct {
	Rows []struct {
		Amount      types.Number `json:"amount"`
		ProductID   string       `json:"productId"`
		Asset       string       `json:"asset"`
		CreateTime  types.Time   `json:"createTime"`
		Type        string       `json:"type"`
		ProductName string       `json:"productName"`
		OrderID     int64        `json:"orderId"`
	} `json:"rows"`
	Total string `json:"total"`
}

// DualInvestmentProduct represents a dual-investment product instance.
type DualInvestmentProduct struct {
	Total int64 `json:"total"`
	List  []struct {
		ID                   string       `json:"id"`
		InvestCoin           string       `json:"investCoin"`
		ExercisedCoin        string       `json:"exercisedCoin"`
		StrikePrice          types.Number `json:"strikePrice"`
		Duration             int64        `json:"duration"`
		SettleDate           int64        `json:"settleDate"`
		PurchaseDecimal      float64      `json:"purchaseDecimal"`
		PurchaseEndTime      types.Time   `json:"purchaseEndTime"`
		CanPurchase          bool         `json:"canPurchase"`
		Apr                  string       `json:"apr"`
		OrderID              int64        `json:"orderId"`
		MinAmount            types.Number `json:"minAmount"`
		MaxAmount            types.Number `json:"maxAmount"`
		CreateTimestamp      types.Time   `json:"createTimestamp"`
		OptionType           string       `json:"optionType"`
		IsAutoCompoundEnable bool         `json:"isAutoCompoundEnable"`
		AutoCompoundPlanList []string     `json:"autoCompoundPlanList"`
	} `json:"list"`
}

// DualInvestmentProductSubscription represents a dual product subscription response
type DualInvestmentProductSubscription struct {
	PositionID         int64        `json:"positionId"`
	InvestCoin         string       `json:"investCoin"`
	ExercisedCoin      string       `json:"exercisedCoin"`
	SubscriptionAmount types.Number `json:"subscriptionAmount"`
	Duration           int64        `json:"duration"`
	AutoCompoundPlan   string       `json:"autoCompoundPlan"`
	StrikePrice        types.Number `json:"strikePrice"`
	SettleDate         int64        `json:"settleDate"`
	PurchaseStatus     string       `json:"purchaseStatus"`
	Apr                string       `json:"apr"`
	OrderID            int64        `json:"orderId"`
	PurchaseTime       types.Time   `json:"purchaseTime"`
	OptionType         string       `json:"optionType"`
}

// DualInvestmentPositions represents a dual investment positions
type DualInvestmentPositions struct {
	Total int `json:"total"`
	List  []struct {
		ID                 string       `json:"id"`
		InvestCoin         string       `json:"investCoin"`
		ExercisedCoin      string       `json:"exercisedCoin"`
		SubscriptionAmount types.Number `json:"subscriptionAmount"`
		StrikePrice        types.Number `json:"strikePrice"`
		Duration           int          `json:"duration"`
		SettleDate         int64        `json:"settleDate"`
		PurchaseStatus     string       `json:"purchaseStatus"`
		Apr                string       `json:"apr"`
		OrderID            int64        `json:"orderId"`
		PurchaseEndTime    types.Time   `json:"purchaseEndTime"`
		OptionType         string       `json:"optionType"`
		AutoCompoundPlan   string       `json:"autoCompoundPlan"`
	} `json:"list"`
}

// DualInvestmentAccount represents a dual investment account
type DualInvestmentAccount struct {
	TotalAmountInBTC  types.Number `json:"totalAmountInBTC"`  // Total BTC amounts in Dual Investment
	TotalAmountInUSDT types.Number `json:"totalAmountInUSDT"` // Total USDT equivalents in BTC in Dual Investment
}

// AutoCompoundStatus represents change auto-compound status
type AutoCompoundStatus struct {
	PositionID       string `json:"positionId"`
	AutoCompoundPlan string `json:"autoCompoundPlan"`
}

// AutoInvestmentAsset represents list target asset info detail.
type AutoInvestmentAsset struct {
	TargetAssets        []string `json:"targetAssets"`
	AutoInvestAssetList []struct {
		TargetAsset             string `json:"targetAsset"`
		RoiAndDimensionTypeList []struct {
			SimulateRoi    string       `json:"simulateRoi"`
			DimensionValue types.Number `json:"dimensionValue"`
			DimensionUnit  string       `json:"dimensionUnit"`
		} `json:"roiAndDimensionTypeList"`
	} `json:"autoInvestAssetList"`
}

// ROIAssetData represents a ROI asset
type ROIAssetData struct {
	Date        string `json:"date"`        // date of the ROI accumulation
	SimulateROI string `json:"simulateRoi"` // value of calculated ROI till the date
}

// AutoInvestAssets represents an auto-invest asset instance.
type AutoInvestAssets struct {
	TargetAssets []string `json:"targetAssets"`
	SourceAssets []string `json:"sourceAssets"`
}

// SourceAssetsList represents source investment assets list/
type SourceAssetsList struct {
	FeeRate      types.Number `json:"feeRate"`
	TaxRate      types.Number `json:"taxRate"`
	SourceAssets []struct {
		SourceAsset    string       `json:"sourceAsset"`
		AssetMinAmount types.Number `json:"assetMinAmount"`
		AssetMaxAmount types.Number `json:"assetMaxAmount"`
		Scale          types.Number `json:"scale"`
		FlexibleAmount types.Number `json:"flexibleAmount"`
	} `json:"sourceAssets"`
}

// InvestmentPlanParams represents a parameter for investment plan creation
type InvestmentPlanParams struct {
	SourceType               string            `json:"sourceType,omitempty"` // "MAIN_SITE" for Binance,“TR” for Binance Turkey
	RequestID                string            `json:"requestId,omitempty"`
	PlanType                 string            `json:"planType,omitempty"` // “SINGLE”,”PORTFOLIO”,”INDEX”
	IndexID                  int64             `json:"indexId,omitempty"`
	SubscriptionAmount       float64           `json:"subscriptionAmount,omitempty"`
	SubscriptionCycle        string            `json:"subscriptionCycle,omitempty"` // "H1", "H4", "H8","H12", "WEEKLY","DAILY","MONTHLY","BI_WEEKLY"
	SubscriptionStartDay     int64             `json:"subscriptionStartDay,omitempty"`
	SubscriptionStartWeekday string            `json:"subscriptionStartWeekday,omitempty"` // “MON”,”TUE”,”WED”,”THU”,”FRI”,”SAT”,”SUN”; Mandatory if “subscriptionCycleNumberUnit” = “WEEKLY” or “BI_WEEKLY”, Must be sent in form of UTC+0
	SubscriptionStartTime    int64             `json:"subscriptionStartTime"`              // “0,1,2,3,4,5,6,7,8,..23”;Must be sent in form of UTC+0
	SourceAsset              currency.Code     `json:"sourceAsset,omitempty"`
	FlexibleAllowedToUse     bool              `json:"flexibleAllowedToUse"`
	Details                  []PortfolioDetail `json:"details,omitempty"`
}

// PortfolioDetail represents a portfolio detail instance.
type PortfolioDetail struct {
	TargetAsset currency.Code `json:"targetAsset"`
	Percentage  int64         `json:"percentage"`
}

// InvestmentPlanResponse represents an investment plan creation response.
type InvestmentPlanResponse struct {
	PlanID                int64      `json:"planId"`
	NextExecutionDateTime types.Time `json:"nextExecutionDateTime"`
}

// AdjustInvestmentPlan represents parameters for investment plan adjustment.
type AdjustInvestmentPlan struct {
	PlanID                   int64             `json:"planId"`
	SubscriptionAmount       float64           `json:"subscriptionAmount,omitempty"`
	SubscriptionCycle        string            `json:"subscriptionCycle,omitempty"`
	SubscriptionStartDay     int64             `json:"subscriptionStartDay,omitempty"`
	SubscriptionStartWeekday string            `json:"subscriptionStartWeekday,omitempty"` // “MON”,”TUE”,”WED”,”THU”,”FRI”,”SAT”,”SUN”; Mandatory if “subscriptionCycleNumberUnit” = “WEEKLY” or “BI_WEEKLY”, Must be sent in form of UTC+0
	SubscriptionStartTime    int64             `json:"subscriptionStartTime"`              // “0,1,2,3,4,5,6,7,8,..23”;Must be sent in form of UTC+0
	SourceAsset              currency.Code     `json:"sourceAsset,omitempty"`
	FlexibleAllowedToUse     bool              `json:"flexibleAllowedToUse"`
	Details                  []PortfolioDetail `json:"-"`
}

// ChangePlanStatusResponse represents a change plan status response.
type ChangePlanStatusResponse struct {
	PlanID                int64      `json:"planId"`
	NextExecutionDateTime types.Time `json:"nextExecutionDateTime"`
	Status                string     `json:"status"`
}

// InvestmentPlans represents an investment plans
type InvestmentPlans struct {
	PlanValueInUSD string `json:"planValueInUSD"`
	PlanValueInBTC string `json:"planValueInBTC"`
	PnlInUSD       string `json:"pnlInUSD"`
	Roi            string `json:"roi"`
	Plans          []struct {
		PlanID                   int64        `json:"planId"`
		PlanType                 string       `json:"planType"`
		EditAllowed              string       `json:"editAllowed"`
		CreationDateTime         types.Time   `json:"creationDateTime"`
		FirstExecutionDateTime   types.Time   `json:"firstExecutionDateTime"`
		NextExecutionDateTime    types.Time   `json:"nextExecutionDateTime"`
		Status                   string       `json:"status"`
		LastUpdatedDateTime      types.Time   `json:"lastUpdatedDateTime"`
		TargetAsset              string       `json:"targetAsset"`
		TotalTargetAmount        types.Number `json:"totalTargetAmount"`
		SourceAsset              string       `json:"sourceAsset"`
		TotalInvestedInUSD       string       `json:"totalInvestedInUSD"`
		SubscriptionAmount       types.Number `json:"subscriptionAmount"`
		SubscriptionCycle        string       `json:"subscriptionCycle"`
		SubscriptionStartDay     string       `json:"subscriptionStartDay"`
		SubscriptionStartWeekday string       `json:"subscriptionStartWeekday"`
		SubscriptionStartTime    types.Time   `json:"subscriptionStartTime"`
		SourceWallet             string       `json:"sourceWallet"`
		FlexibleAllowedToUse     string       `json:"flexibleAllowedToUse"`
		PlanValueInUSD           string       `json:"planValueInUSD"`
		PnlInUSD                 string       `json:"pnlInUSD"`
		ROI                      string       `json:"roi"`
	} `json:"plans"`
}

// InvestmentPlanHoldingDetail represents a holding detail of an investment plan.
type InvestmentPlanHoldingDetail struct {
	PlanID                 int64      `json:"planId"`
	PlanType               string     `json:"planType"`
	EditAllowed            string     `json:"editAllowed"`
	FlexibleAllowedToUse   string     `json:"flexibleAllowedToUse"`
	CreationDateTime       types.Time `json:"creationDateTime"`
	FirstExecutionDateTime types.Time `json:"firstExecutionDateTime"`
	NextExecutionDateTime  types.Time `json:"nextExecutionDateTime"`
	Status                 string     `json:"status"`
	TargetAsset            string     `json:"targetAsset"`
	SourceAsset            string     `json:"sourceAsset"`
	PlanValueInUSD         string     `json:"planValueInUSD"`
	PnlInUSD               string     `json:"pnlInUSD"`
	Roi                    string     `json:"roi"`
	TotalInvestedInUSD     string     `json:"totalInvestedInUSD"`
	Details                []struct {
		TargetAsset         string       `json:"targetAsset"`
		AveragePriceInUSD   string       `json:"averagePriceInUSD"`
		TotalInvestedInUSD  string       `json:"totalInvestedInUSD"`
		PurchasedAmount     types.Number `json:"purchasedAmount"`
		PurchasedAmountUnit string       `json:"purchasedAmountUnit"`
		PnlInUSD            string       `json:"pnlInUSD"`
		ROI                 string       `json:"roi"`
		Percentage          string       `json:"percentage"`
		AssetStatus         string       `json:"assetStatus"`
		AvailableAmount     types.Number `json:"availableAmount"`
		AvailableAmountUnit string       `json:"availableAmountUnit"`
		RedeemedAmout       types.Number `json:"redeemedAmout"`
		RedeemedAmoutUnit   string       `json:"redeemedAmoutUnit"`
		AssetValueInUSD     string       `json:"assetValueInUSD"`
	} `json:"details"`
}

// AutoInvestSubscriptionTransactionItem represents subscription transaction item
type AutoInvestSubscriptionTransactionItem struct {
	ID                  int          `json:"id"`
	TargetAsset         string       `json:"targetAsset"`
	ExecutionType       string       `json:"executionType"` // ONE_TIME,RECURRING
	PlanType            string       `json:"planType"`
	PlanName            string       `json:"planName"`
	PlanID              int64        `json:"planId"`
	TransactionDateTime types.Time   `json:"transactionDateTime"`
	TransactionStatus   string       `json:"transactionStatus"`
	FailedType          string       `json:"failedType"`
	SourceAsset         string       `json:"sourceAsset"`
	SourceAssetAmount   types.Number `json:"sourceAssetAmount"`
	TargetAssetAmount   types.Number `json:"targetAssetAmount"`
	SourceWallet        string       `json:"sourceWallet"`
	FlexibleUsed        string       `json:"flexibleUsed"` // whether simple earn wallet is used
	TransactionFee      string       `json:"transactionFee"`
	TransactionFeeUnit  string       `json:"transactionFeeUnit"` // denominated coin of the transaction fee
	ExecutionPrice      types.Number `json:"executionPrice"`     // price of the subscription price. It's amount of source asset equivalent of 1 unit of target asset
	SubscriptionCycle   types.Number `json:"subscriptionCycle"`
}

// AutoInvestSubscriptionTransactionResponse represents a detail of auto-investment subscription transaction.
type AutoInvestSubscriptionTransactionResponse struct {
	Total int64                                   `json:"total"`
	List  []AutoInvestSubscriptionTransactionItem `json:"list"`
}

// AutoInvestmentIndexDetail represents an index detail information.
type AutoInvestmentIndexDetail struct {
	IndexID         int64  `json:"indexId"`
	IndexName       string `json:"indexName"`
	Status          string `json:"status"`
	AssetAllocation []struct {
		TargetAsset string `json:"targetAsset"`
		Allocation  string `json:"allocation"`
	} `json:"assetAllocation"`
}

// IndexLinkedPlanPositionDetail represents an index linked investment-plan positions detail.
type IndexLinkedPlanPositionDetail struct {
	IndexID              int64        `json:"indexId"`
	TotalInvestedInUSD   types.Number `json:"totalInvestedInUSD"`
	CurrentInvestedInUSD types.Number `json:"currentInvestedInUSD"`
	PnlInUSD             types.Number `json:"pnlInUSD"`
	ROI                  types.Number `json:"roi"`
	AssetAllocation      []struct {
		TargetAsset string `json:"targetAsset"`
		Allocation  string `json:"allocation"`
	} `json:"assetAllocation"`
	Details []struct {
		TargetAsset          string       `json:"targetAsset"`
		AveragePriceInUSD    types.Number `json:"averagePriceInUSD"`
		TotalInvestedInUSD   types.Number `json:"totalInvestedInUSD"`
		CurrentInvestedInUSD types.Number `json:"currentInvestedInUSD"`
		PurchasedAmount      types.Number `json:"purchasedAmount"`
		PNLInUSD             types.Number `json:"pnlInUSD"`
		ROI                  types.Number `json:"roi"`
		Percentage           types.Number `json:"percentage"`
		AvailableAmount      types.Number `json:"availableAmount"`
		RedeemedAmount       types.Number `json:"redeemedAmount"`
		AssetValueInUSD      types.Number `json:"assetValueInUSD"`
	} `json:"details"`
}

// OneTimeTransactionParams request parameters for one-time transaction instance.
type OneTimeTransactionParams struct {
	SourceType           string            `json:"sourceType"` // "MAIN_SITE" for Binance,“TR” for Binance Turkey
	RequestID            string            `json:"requestId"`  // if not null, must follow sourceType + unique string, e.g: TR12354859
	SubscriptionAmount   float64           `json:"subscriptionAmount"`
	SourceAsset          currency.Code     `json:"sourceAsset"`
	FlexibleAllowedToUse bool              `json:"flexibleAllowedToUse"` // true/false；true: using flexible wallet
	PlanID               int64             `json:"planId,omitempty"`     // PORTFOLIO plan's Id
	IndexID              int64             `json:"indexId,omitempty"`
	Details              []PortfolioDetail `json:"-"` // sum(all node's percentage) == 100，sum(all node's percentage) == 100， When input request parameter, each entry should be like details[0].targetAsset=BTC, Example of the request parameter array:
}

// OneTimeTransactionResponse represents a response data for one-time transaction
type OneTimeTransactionResponse struct {
	TransactionID int64 `json:"transactionId"`
	WaitSecond    int64 `json:"waitSecond"`
}

// PlanRedemption represents an index plan redemption transaction instance.
type PlanRedemption struct {
	IndexID            int64        `json:"indexId"`
	IndexName          string       `json:"indexName"`
	RedemptionID       int64        `json:"redemptionId"`
	Status             string       `json:"status"`
	Asset              string       `json:"asset"`
	Amount             types.Number `json:"amount"`
	RedemptionDateTime types.Time   `json:"redemptionDateTime"`
	TransactionFee     types.Number `json:"transactionFee"`
	TransactionFeeUnit string       `json:"transactionFeeUnit"`
}

// IndexLinkedPlanRebalanceDetail represents an index plan rebalance instance detail.
type IndexLinkedPlanRebalanceDetail struct {
	IndexID            int64        `json:"indexId"`
	IndexName          string       `json:"indexName"`
	RebalanceID        int64        `json:"rebalanceId"`
	Status             string       `json:"status"` // rebalance status  SUCCESS/INIT
	RebalanceFee       types.Number `json:"rebalanceFee"`
	RebalanceFeeUnit   string       `json:"rebalanceFeeUnit"`
	TransactionDetails []struct {
		Asset               string       `json:"asset"`               // assets to be rebalanced
		TransactionDateTime types.Time   `json:"transactionDateTime"` // rebalance transaction timestamp
		RebalanceDirection  string       `json:"rebalanceDirection"`  // rebalance direction
		RebalanceAmount     types.Number `json:"rebalanceAmount"`     // rebalance amount for the asset
	} `json:"transactionDetails"`
}

// StakingSubscriptionResponse represents V2 staking subscription response.
type StakingSubscriptionResponse struct {
	Success         bool         `json:"success"`
	WbethAmount     types.Number `json:"wbethAmount"`
	ConversionRatio string       `json:"conversionRatio"`
}

// StakingRedemptionResponse represents redemption response response.
type StakingRedemptionResponse struct {
	Success         bool         `json:"success"`
	ArrivalTime     types.Time   `json:"arrivalTime"`
	EthAmount       types.Number `json:"ethAmount"`
	ConversionRatio types.Number `json:"conversionRatio"`
}

// ETHStakingHistory represents ETH staking history
type ETHStakingHistory struct {
	Rows []struct {
		Time             types.Time   `json:"time"`
		Asset            string       `json:"asset"`
		Amount           types.Number `json:"amount"`
		Status           string       `json:"status"`
		DistributeAmount types.Number `json:"distributeAmount"`
		ConversionRatio  types.Number `json:"conversionRatio"`
	} `json:"rows"`
	Total int64 `json:"total"`
}

// ETHRedemptionHistory represents ETH redemption history
type ETHRedemptionHistory struct {
	Rows []struct {
		Time             types.Time   `json:"time"`
		ArrivalTime      types.Time   `json:"arrivalTime"`
		Asset            string       `json:"asset"`
		Amount           types.Number `json:"amount"`
		Status           string       `json:"status"` // PENDING,SUCCESS,FAILED
		DistributeAsset  string       `json:"distributeAsset"`
		DistributeAmount types.Number `json:"distributeAmount"`
		ConversionRatio  types.Number `json:"conversionRatio"`
	} `json:"rows"`
	Total int64 `json:"total"`
}

// BETHRewardDistribution represents a BETH reward distribution history
type BETHRewardDistribution struct {
	Rows []struct {
		Time                 types.Time   `json:"time"`
		Asset                string       `json:"asset"`
		Holding              string       `json:"holding"`              // BETH holding balance
		Amount               types.Number `json:"amount"`               // Distributed rewards
		AnnualPercentageRate string       `json:"annualPercentageRate"` // 0.5 means 50% here
		Status               string       `json:"status"`
	} `json:"rows"`
	Total int64 `json:"total"`
}

// ETHStakingQuota represents an ETH current staking quota response.
type ETHStakingQuota struct {
	LeftStakingPersonalQuota    types.Number `json:"leftStakingPersonalQuota"`
	LeftRedemptionPersonalQuota types.Number `json:"leftRedemptionPersonalQuota"`
}

// WBETHRateHistory represents a WBETH rate history
type WBETHRateHistory struct {
	Rows []struct {
		AnnualPercentageRate types.Number `json:"annualPercentageRate"`
		ExchangeRate         types.Number `json:"exchangeRate"`
		Time                 types.Time   `json:"time"`
	} `json:"rows"`
	Total string `json:"total"`
}

// ETHStakingAccountDetail represents ETH staking account detail.
type ETHStakingAccountDetail struct {
	CumulativeProfitInBETH types.Number `json:"cumulativeProfitInBETH"`
	LastDayProfitInBETH    types.Number `json:"lastDayProfitInBETH"`
}

// StakingAccountV2Response represents ETH staking account detail
type StakingAccountV2Response struct {
	HoldingInETH string `json:"holdingInETH"`
	Holdings     struct {
		WbethAmount types.Number `json:"wbethAmount"`
		BethAmount  types.Number `json:"bethAmount"`
	} `json:"holdings"`
	ThirtyDaysProfitInETH string `json:"thirtyDaysProfitInETH"`
	Profit                struct {
		AmountFromWBETH types.Number `json:"amountFromWBETH"` // Profit accrued within WBETH
		AmountFromBETH  types.Number `json:"amountFromBETH"`  // BETH distributed to your Spot Wallet
	} `json:"profit"`
}

// WrapBETHResponse wrap BETH response.
type WrapBETHResponse struct {
	Success      bool         `json:"success"`
	WbethAmount  types.Number `json:"wbethAmount"`
	ExchangeRate types.Number `json:"exchangeRate"`
}

// WBETHWrapHistory represents a BETH wrap/unwrap history
type WBETHWrapHistory struct {
	Rows []struct {
		Time         types.Time   `json:"time"`
		FromAsset    string       `json:"fromAsset"`
		FromAmount   types.Number `json:"fromAmount"`
		ToAsset      string       `json:"toAsset"`
		ToAmount     types.Number `json:"toAmount"`
		ExchangeRate types.Number `json:"exchangeRate"` // BETH amount per 1 WBETH
		Status       string       `json:"status"`       // PENDING,SUCCESS,FAILED
	} `json:"rows"`
	Total int64 `json:"total"`
}

// WBETHRewardHistory represents a WBETH reward history item.
type WBETHRewardHistory struct {
	EstRewardsInETH string `json:"estRewardsInETH"`
	Rows            []struct {
		Time                 types.Time   `json:"time"`
		AmountInETH          types.Number `json:"amountInETH"` // Estimated rewards accrued within WBETH
		Holding              types.Number `json:"holding"`     // WBETH holding balance
		HoldingInETH         types.Number `json:"holdingInETH"`
		AnnualPercentageRate types.Number `json:"annualPercentageRate"`
	} `json:"rows"`
	Total int64 `json:"total"`
}

// AlgorithmsList represents list of mining algorithms.
type AlgorithmsList struct {
	Code int64  `json:"code"`
	Msg  string `json:"msg"`
	Data []struct {
		AlgoName  string `json:"algoName"`
		AlgoID    int    `json:"algoId"`    // Algorithm ID
		PoolIndex int    `json:"poolIndex"` // Sequence
		Unit      string `json:"unit"`
	} `json:"data"`
}

// CoinNames represents coins and corresponding algorithms used
type CoinNames struct {
	Code int64  `json:"code"`
	Msg  string `json:"msg"`
	Data []struct {
		CoinName  string `json:"coinName"`
		CoinID    int64  `json:"coinId"`
		PoolIndex int64  `json:"poolIndex"`
		AlgoID    int64  `json:"algoId"`
		AlgoName  string `json:"algoName"`
	} `json:"data"`
}

// MinersDetailList represents list of miners and their detail
type MinersDetailList struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data []struct {
		WorkerName    string `json:"workerName"` // Mining Account name
		Type          string `json:"type"`       // Type of hourly hashrate
		HashrateDatas []struct {
			Time     types.Time `json:"time"`
			Hashrate string     `json:"hashrate"`
			Reject   int64      `json:"reject"` // Rejection Rate
		} `json:"hashrateDatas"`
	} `json:"data"`
}

// MinerLists represents list of miners
type MinerLists struct {
	Code int64  `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		WorkerDatas []struct {
			WorkerID      string     `json:"workerId"`
			WorkerName    string     `json:"workerName"`
			Status        int64      `json:"status"`
			HashRate      int64      `json:"hashRate"`
			DayHashRate   int64      `json:"dayHashRate"`
			RejectRate    int64      `json:"rejectRate"`
			LastShareTime types.Time `json:"lastShareTime"`
		} `json:"workerDatas"`
		TotalNum int64 `json:"totalNum"`
		PageSize int64 `json:"pageSize"`
	} `json:"data"`
}

// EarningList represents list of mining payments list
type EarningList struct {
	Code int64  `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		AccountProfits []struct {
			Time           types.Time `json:"time"`
			Type           int64      `json:"type"`
			HashTransfer   float64    `json:"hashTransfer"`
			TransferAmount float64    `json:"transferAmount"`
			DayHashRate    int64      `json:"dayHashRate"`
			ProfitAmount   float64    `json:"profitAmount"`
			CoinName       string     `json:"coinName"`
			Status         int        `json:"status"`
		} `json:"accountProfits"`
		TotalNum int64 `json:"totalNum"`
		PageSize int64 `json:"pageSize"`
	} `json:"data"`
}

// ExtraBonus represents an extra bonus list information.
type ExtraBonus struct {
	Code int64  `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		OtherProfits []struct {
			Time         types.Time `json:"time"`
			CoinName     string     `json:"coinName"`
			Type         int64      `json:"type"`
			ProfitAmount float64    `json:"profitAmount"`
			Status       int64      `json:"status"`
		} `json:"otherProfits"`
		TotalNum int64 `json:"totalNum"`
		PageSize int64 `json:"pageSize"`
	} `json:"data"`
}

// HashrateHashTransfers represents a hashrate rescale list.
type HashrateHashTransfers struct {
	Code int64  `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		ConfigDetails []struct {
			ConfigID       int64      `json:"configId"`       // Mining ID
			PoolUsername   string     `json:"poolUsername"`   // Transfer out of subaccount
			ToPoolUsername string     `json:"toPoolUsername"` // Transfer into subaccount
			AlgoName       string     `json:"algoName"`       // Transfer algorithm
			HashRate       int64      `json:"hashRate"`       // Transferred Hashrate quantity
			StartDay       types.Time `json:"startDay"`
			EndDay         types.Time `json:"endDay"`
			Status         int64      `json:"status"` // Status：0 Processing，1：Cancelled，2：Terminated
		} `json:"configDetails"`
		TotalNum int64 `json:"totalNum"`
		PageSize int64 `json:"pageSize"`
	} `json:"data"`
}

// HashrateRescaleDetail represents  a hashrate rescale detail
type HashrateRescaleDetail struct {
	Code int64  `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		ProfitTransferDetails []struct {
			PoolUsername   string  `json:"poolUsername"`   // Transfer out of sub-account
			ToPoolUsername string  `json:"toPoolUsername"` // Transfer into subaccount
			AlgoName       string  `json:"algoName"`       // Transfer algorithm
			HashRate       int64   `json:"hashRate"`       // Transferred Hashrate quantity
			Day            int     `json:"day"`
			Amount         float64 `json:"amount"`
			CoinName       string  `json:"coinName"`
		} `json:"profitTransferDetails"`
		TotalNum int64 `json:"totalNum"`
		PageSize int64 `json:"pageSize"`
	} `json:"data"`
}

// HashrateRescalResponse represents a response for hashrate rescale request
type HashrateRescalResponse struct {
	Code int64  `json:"code"`
	Msg  string `json:"msg"`
	Data int64  `json:"data"`
}

// UserStatistics represents user mining statistics
type UserStatistics struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		FifteenMinHashRate string                  `json:"fifteenMinHashRate"`
		DayHashRate        string                  `json:"dayHashRate"`
		ValidNum           int64                   `json:"validNum"`
		InvalidNum         int64                   `json:"invalidNum"`
		ProfitToday        map[string]types.Number `json:"profitToday"`
		ProfitYesterday    map[string]types.Number `json:"profitYesterday"`
		UserName           string                  `json:"userName"`
		Unit               string                  `json:"unit"`
		Algo               string                  `json:"algo"`
	} `json:"data"`
}

// MiningAccounts represents a mining account list
type MiningAccounts struct {
	Code int64  `json:"code"`
	Msg  string `json:"msg"`
	Data []struct {
		Type     string `json:"type"`     // Type of hourly hashrate. eg. H_hashrate: hourly hash rate, D_hashrate: ...
		UserName string `json:"userName"` // Mining account
		List     []struct {
			Time     types.Time   `json:"time"`
			Hashrate types.Number `json:"hashrate"`
			Reject   string       `json:"reject"` // Rejection Rate
		} `json:"list"`
	} `json:"data"`
}

// MiningAccountEarnings represents a mining account earning details.
type MiningAccountEarnings struct {
	Code int64  `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		AccountProfits []struct {
			Time         types.Time `json:"time"`
			CoinName     string     `json:"coinName"`
			Type         int64      `json:"type"` // 0:Referral 1：Refund 2：Rebate
			SubAccountID int64      `json:"puid"`
			SubName      string     `json:"subName"` // Mining account
			Amount       float64    `json:"amount"`
		} `json:"accountProfits"`
		TotalNum int64 `json:"totalNum"`
		PageSize int64 `json:"pageSize"`
	} `json:"data"`
}

// FundTransferResponse represents a transfer response.
type FundTransferResponse struct {
	TransferID int64 `json:"transId"`
}

// FutureFundTransfers represents list of fund transfers between spot and futures accounts.
type FutureFundTransfers struct {
	Rows []struct {
		Asset     string       `json:"asset"`
		TranID    int64        `json:"tranId"`
		Amount    types.Number `json:"amount"`
		Type      string       `json:"type"`
		Timestamp types.Time   `json:"timestamp"`
		Status    string       `json:"status"`
	} `json:"rows"`
	Total int64 `json:"total"`
}

// HistoricalOrderbookDownloadLink represents a download link information for historical orderbook data.
type HistoricalOrderbookDownloadLink struct {
	Data []struct {
		Day string `json:"day"`
		URL string `json:"url"`
	} `json:"data"`
}

// VolumeParticipationOrderParams represents a volume participation new order.
type VolumeParticipationOrderParams struct {
	Symbol       string  `json:"symbol"`
	Side         string  `json:"side"`                   // Trading side ( BUY or SELL )
	PositionSide string  `json:"positionSide,omitempty"` // Default BOTH for One-way Mode ; LONG or SHORT for Hedge Mode. It must be sent in Hedge Mode.
	Quantity     float64 `json:"quantity"`               // Quantity of base asset; The notional (quantity * mark price(base asset)) must be more than the equivalent of 1,000 USDT and less than the equivalent of 1,000,000 USDT
	Urgency      string  `json:"urgency"`                // Represent the relative speed of the current execution; ENUM: LOW, MEDIUM, HIGH
	ClientAlgoID string  `json:"clientAlgoId,omitempty"`
	ReduceOnly   bool    `json:"reduceOnly,omitempty"`
	LimitPrice   float64 `json:"limitPrice,omitempty"` // Limit price of the order; If it is not sent, will place order by market price by default
}

// TWAPOrderParams represents a time-weighted average price(TWAP) order parameters.
type TWAPOrderParams struct {
	Symbol       string  `json:"symbol"`
	Side         string  `json:"side"`
	PositionSide string  `json:"positionSide,omitempty"`
	Quantity     float64 `json:"quantity"`
	Duration     int64   `json:"duration"`
	ClientAlgoID string  `json:"clientAlgoId,omitempty"`
	ReduceOnly   bool    `json:"reduceOnly,omitempty"`
	LimitPrice   float64 `json:"limitPrice,omitempty"`
}

// AlgoOrderResponse represents a response after placing structure of TWAP and VP, and cancelling algo order
type AlgoOrderResponse struct {
	ClientAlgoID string `json:"clientAlgoId"`
	Success      bool   `json:"success"`
	Code         int64  `json:"code"`
	Msg          string `json:"msg"`

	// AlgoID used when cancelling an algo order.
	AlgoID int64 `json:"algoId"`
}

// AlgoOrders represents an algo order instance.
type AlgoOrders struct {
	Total  int64 `json:"total"`
	Orders []struct {
		AlgoID       int64        `json:"algoId"`
		Symbol       string       `json:"symbol"`
		Side         string       `json:"side"`
		PositionSide string       `json:"positionSide"`
		TotalQty     types.Number `json:"totalQty"`
		ExecutedQty  types.Number `json:"executedQty"`
		ExecutedAmt  types.Number `json:"executedAmt"`
		AvgPrice     types.Number `json:"avgPrice"`
		ClientAlgoID string       `json:"clientAlgoId"`
		BookTime     types.Time   `json:"bookTime"`
		EndTime      types.Time   `json:"endTime"`
		AlgoStatus   string       `json:"algoStatus"`
		AlgoType     string       `json:"algoType"`
		Urgency      string       `json:"urgency"`
	} `json:"orders"`
}

// AlgoSubOrders represents an algo sub-order
type AlgoSubOrders struct {
	Total       int64        `json:"total"`
	ExecutedQty types.Number `json:"executedQty"`
	ExecutedAmt types.Number `json:"executedAmt"`
	SubOrders   []struct {
		AlgoID      int64        `json:"algoId"`
		OrderID     int64        `json:"orderId"`
		OrderStatus string       `json:"orderStatus"`
		ExecutedQty types.Number `json:"executedQty"`
		ExecutedAmt types.Number `json:"executedAmt"`
		FeeAmt      types.Number `json:"feeAmt"`
		FeeAsset    string       `json:"feeAsset"`
		BookTime    types.Time   `json:"bookTime"`
		AvgPrice    types.Number `json:"avgPrice"`
		Side        string       `json:"side"`
		Symbol      string       `json:"symbol"`
		SubID       int64        `json:"subId"`
		TimeInForce string       `json:"timeInForce"`
		OrigQty     types.Number `json:"origQty"`
	} `json:"subOrders"`
}

// SpotTWAPOrderParam represents a spot time-weighted averaged price(TWAP) order params.
type SpotTWAPOrderParam struct {
	Symbol       string  `json:"symbol"`
	Side         string  `json:"side"`
	Quantity     float64 `json:"quantity"`
	Duration     int64   `json:"duration"`
	ClientAlgoID string  `json:"clientAlgoId,omitempty"`
	LimitPrice   float64 `json:"limitPrice,omitempty"`
	STPMode      string  `json:"stpMode,omitempty"`
}

// ClassicPMAccountInfo represents a classic portfolio margin account information.
type ClassicPMAccountInfo struct {
	UniMMR             string `json:"uniMMR"`             // Classic Portfolio margin account maintenance margin rate
	AccountEquity      string `json:"accountEquity"`      // Account equity, unit：USD
	ActualEquity       string `json:"actualEquity"`       // Actual equity, unit：USD
	AccountMaintMargin string `json:"accountMaintMargin"` // Classic Portfolio margin account maintenance margin, unit：USD
	AccountStatus      string `json:"accountStatus"`      // Classic Portfolio margin account status:"NORMAL", "MARGIN_CALL", "SUPPLY_MARGIN", "REDUCE_ONLY", "ACTIVE_LIQUIDATION", "FORCE_LIQUIDAT
	AccountType        string `json:"accountType"`        // PM_1 for classic PM, PM_2 for PM
}

// PMCollateralRate represents a classic portfolio margin collateral rate
type PMCollateralRate struct {
	Asset          string       `json:"asset"`
	CollateralRate types.Number `json:"collateralRate"`
}

// PMBankruptacyLoanAmount represents a classic portfolio margin bankruptcy loan amount
type PMBankruptacyLoanAmount struct {
	Asset  string       `json:"asset"`
	Amount types.Number `json:"amount"` // portfolio margin bankruptcy loan amount in BUSD
}

// PMNegativeBalaceInterestHistory represents a portfolio margin negative balance interest history.
type PMNegativeBalaceInterestHistory struct {
	Asset               string       `json:"asset"`
	Interest            types.Number `json:"interest"` // interest amount
	InterestAccruedTime types.Time   `json:"interestAccruedTime"`
	InterestRate        types.Number `json:"interestRate"` // daily interest rate
	Principal           string       `json:"principal"`
}

// PMIndexPrice represents PM asset index price
type PMIndexPrice struct {
	Asset           string       `json:"asset"`
	AssetIndexPrice types.Number `json:"assetIndexPrice"`
	Time            types.Time   `json:"time"`
}

// FundAutoCollectionResponse represents futures Account to Margin account transfer response.
type FundAutoCollectionResponse struct {
	Message string `json:"msg"`
}

// PMAssetLeverage represents an asset leverage
type PMAssetLeverage struct {
	Asset    string `json:"asset"`
	Leverage int64  `json:"leverage"`
}

// BLVTTokenDetail represents a binance leverage token detail
type BLVTTokenDetail struct {
	TokenName      string `json:"tokenName"`
	Description    string `json:"description"`
	Underlying     string `json:"underlying"`
	TokenIssued    string `json:"tokenIssued"`
	Basket         string `json:"basket"`
	CurrentBaskets []struct {
		Symbol        string       `json:"symbol"`
		Amount        types.Number `json:"amount"`
		NotionalValue string       `json:"notionalValue"`
	} `json:"currentBaskets"`
	Nav                string       `json:"nav"`
	RealLeverage       types.Number `json:"realLeverage"`
	FundingRate        types.Number `json:"fundingRate"`
	DailyManagementFee types.Number `json:"dailyManagementFee"`
	PurchaseFeePct     types.Number `json:"purchaseFeePct"`
	DailyPurchaseLimit types.Number `json:"dailyPurchaseLimit"`
	RedeemFeePct       types.Number `json:"redeemFeePct"`
	DailyRedeemLimit   types.Number `json:"dailyRedeemLimit"`
	Timestamp          types.Time   `json:"timestamp"`
}

// BLVTSubscriptionResponse represents a subscription to BLVT token
type BLVTSubscriptionResponse struct {
	ID        int64        `json:"id"`
	Status    string       `json:"status"` // S, P, and F for "success", "pending", and "failure"
	TokenName string       `json:"tokenName"`
	Amount    types.Number `json:"amount"` // subscribed token amount
	Cost      types.Number `json:"cost"`   // subscription cost in usdt
	Timestamp types.Time   `json:"timestamp"`
}

// BLVTTokenSubscriptionItem represents a subscription instances for BLVT token name.
type BLVTTokenSubscriptionItem struct {
	ID          int64        `json:"id"`
	TokenName   string       `json:"tokenName"`
	Amount      types.Number `json:"amount"`
	Nav         string       `json:"nav"`
	Fee         types.Number `json:"fee"`
	TotalCharge string       `json:"totalCharge"`
	Timestamp   types.Time   `json:"timestamp"`
}

// BLVTRedemption represents a BLVT redemption response.
type BLVTRedemption struct {
	ID           int64        `json:"id"`
	Status       string       `json:"status"`
	TokenName    string       `json:"tokenName"`
	RedeemAmount types.Number `json:"redeemAmount"`
	Amount       types.Number `json:"amount"`
	Timestamp    types.Time   `json:"timestamp"`
}

// BLVTRedemptionItem represents a BLVT redemption record item.
type BLVTRedemptionItem struct {
	ID         int64        `json:"id"`
	TokenName  string       `json:"tokenName"`
	Amount     types.Number `json:"amount"`     // Redemption amount
	Nav        string       `json:"nav"`        // NAV of redemption
	Fee        types.Number `json:"fee"`        // Reemption fee
	NetProceed string       `json:"netProceed"` // Net redemption value in usdt
	Timestamp  types.Time   `json:"timestamp"`
}

// BLVTUserLimitInfo represents a BLVT user limit information.
type BLVTUserLimitInfo struct {
	TokenName                       string       `json:"tokenName"`
	UserDailyTotalPurchaseLimitUSDT types.Number `json:"userDailyTotalPurchaseLimit"`
	UserDailyTotalRedeemLimitUSDT   types.Number `json:"userDailyTotalRedeemLimit"`
}

// FiatTransactionHistory represents a withdrawal and deposit history for fiat currencies.
type FiatTransactionHistory struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    []struct {
		OrderNo         string       `json:"orderNo"`
		FiatCurrency    string       `json:"fiatCurrency"`
		IndicatedAmount types.Number `json:"indicatedAmount"`
		Amount          types.Number `json:"amount"`
		TotalFee        types.Number `json:"totalFee"` // Trade fee
		Method          string       `json:"method"`   // Trade method
		Status          string       `json:"status"`   // Processing, Failed, Successful, Finished, Refunding, Refunded, Refund Failed, Order Partial credit Stopped
		CreateTime      types.Time   `json:"createTime"`
		UpdateTime      types.Time   `json:"updateTime"`
	} `json:"data"`
	Total   int64 `json:"total"`
	Success bool  `json:"success"`
}

// FiatPaymentHistory represents a fiat payments history
type FiatPaymentHistory struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    []struct {
		OrderNo        string       `json:"orderNo"`
		SourceAmount   types.Number `json:"sourceAmount"`   // Fiat trade amount
		FiatCurrency   string       `json:"fiatCurrency"`   // Fiat token
		ObtainAmount   types.Number `json:"obtainAmount"`   // Crypto trade amount
		CryptoCurrency string       `json:"cryptoCurrency"` // Crypto token
		TotalFee       types.Number `json:"totalFee"`       // Trade fee
		Price          types.Number `json:"price"`
		Status         string       `json:"status"` // Processing, Completed, Failed, Refunded
		PaymentMethod  string       `json:"paymentMethod"`
		CreateTime     types.Time   `json:"createTime"`
		UpdateTime     types.Time   `json:"updateTime"`
	} `json:"data"`
	Total   int64 `json:"total"`
	Success bool  `json:"success"`
}

// C2CTransaction represents a C2C transaction history
type C2CTransaction struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    []struct {
		OrderNumber         string       `json:"orderNumber"`
		AdvNo               string       `json:"advNo"`
		TradeType           string       `json:"tradeType"`
		Asset               string       `json:"asset"`
		Fiat                string       `json:"fiat"`
		FiatSymbol          string       `json:"fiatSymbol"`
		Amount              types.Number `json:"amount"` // Quantity (in Crypto)
		TotalPrice          types.Number `json:"totalPrice"`
		UnitPrice           types.Number `json:"unitPrice"`   // Unit Price (in Fiat)
		OrderStatus         string       `json:"orderStatus"` // possible values are: 'PENDING', 'TRADING', 'BUYER_PAYED', 'DISTRIBUTING', 'COMPLETED', 'IN_APPEAL', 'CANCELLED', 'CANCELLED_BY_SYSTEM'
		CreateTime          types.Time   `json:"createTime"`
		Commission          string       `json:"commission"` // Transaction Fee (in Crypto)
		CounterPartNickName string       `json:"counterPartNickName"`
		AdvertisementRole   string       `json:"advertisementRole"`
	} `json:"data"`
	Total   int64 `json:"total"`
	Success bool  `json:"success"`
}

// VIPLoanOngoingOrders represents a VIP loan orders history
type VIPLoanOngoingOrders struct {
	Rows []struct {
		OrderID                          int64        `json:"orderId"`
		LoanCoin                         string       `json:"loanCoin"`
		TotalDebt                        types.Number `json:"totalDebt"`
		LoanRate                         types.Number `json:"loanRate"`
		ResidualInterest                 string       `json:"residualInterest"`
		CollateralAccountID              string       `json:"collateralAccountId"`
		CollateralCoin                   string       `json:"collateralCoin"`
		TotalCollateralValueAfterHaircut string       `json:"totalCollateralValueAfterHaircut"`
		LockedCollateralValue            string       `json:"lockedCollateralValue"`
		CurrentLTV                       string       `json:"currentLTV"`
		ExpirationTime                   types.Time   `json:"expirationTime"`
		LoanDate                         string       `json:"loanDate"`
		LoanTerm                         string       `json:"loanTerm"`
		InitialLtv                       string       `json:"initialLtv"`
		MarginCallLtv                    string       `json:"marginCallLtv"`
		LiquidationLtv                   string       `json:"liquidationLtv"`
	} `json:"rows"`
	Total int64 `json:"total"`
}

// VIPLoanRepayResponse represents a response for VIP loan repayment.
type VIPLoanRepayResponse struct {
	LoanCoin           string       `json:"loanCoin"`
	RepayAmount        types.Number `json:"repayAmount"`
	RemainingPrincipal string       `json:"remainingPrincipal"`
	RemainingInterest  string       `json:"remainingInterest"`
	CollateralCoin     string       `json:"collateralCoin"`
	CurrentLTV         string       `json:"currentLTV"`
	RepayStatus        string       `json:"repayStatus"`
}

// WalletAssetCosts represents a wallet asset cost list.
type WalletAssetCosts []map[string]types.Number

// UnmarshalJSON deserializes a wallet asset cost object or list of objects into slice of map.
func (a *WalletAssetCosts) UnmarshalJSON(data []byte) error {
	var resp []map[string]types.Number
	err := json.Unmarshal(data, &resp)
	if err != nil {
		var singleObj map[string]types.Number
		err = json.Unmarshal(data, &singleObj)
		if err != nil {
			return err
		}
		resp = append(resp, singleObj)
	}
	*a = resp
	return nil
}

// PayTradeHistory represents a pay transactions.
type PayTradeHistory struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    []struct {
		OrderType       string       `json:"orderType"`
		TransactionID   string       `json:"transactionId"`
		TransactionTime types.Time   `json:"transactionTime"`
		Amount          types.Number `json:"amount"`
		Currency        string       `json:"currency"`
		WalletType      int64        `json:"walletType"`
		WalletTypes     []string     `json:"walletTypes"`
		FundsDetail     []struct {
			Currency        string           `json:"currency"`
			Amount          types.Number     `json:"amount"`
			WalletAssetCost WalletAssetCosts `json:"walletAssetCost"`
		} `json:"fundsDetail"`
		PayerInfo struct {
			Name      string `json:"name"`
			Type      string `json:"type"`
			BinanceID int64  `json:"binanceId"`
			AccountID int64  `json:"accountId"`
		} `json:"payerInfo"`
		ReceiverInfo struct {
			Name        string `json:"name"`
			Type        string `json:"type"`
			Email       string `json:"email"`
			BinanceID   int64  `json:"binanceId"`
			AccountID   int64  `json:"accountId"`
			CountryCode string `json:"countryCode"`
			PhoneNumber string `json:"phoneNumber"`
			MobileCode  string `json:"mobileCode"`
			Extend      struct {
				InstitutionName string `json:"institutionName"`
				CardNumber      string `json:"cardNumber"`
				DigitalWalletID string `json:"digitalWalletId"`
			} `json:"extend"`
		} `json:"receiverInfo"`
	} `json:"data"`
	Success bool `json:"success"`
}

// ConvertPairInfo represents a convert-pair information.
type ConvertPairInfo struct {
	FromAsset          string       `json:"fromAsset"`
	ToAsset            string       `json:"toAsset"`
	FromAssetMinAmount types.Number `json:"fromAssetMinAmount"`
	FromAssetMaxAmount types.Number `json:"fromAssetMaxAmount"`
	ToAssetMinAmount   types.Number `json:"toAssetMinAmount"`
	ToAssetMaxAmount   types.Number `json:"toAssetMaxAmount"`
}

// OrderQuantityPrecision represents asset’s precision information
type OrderQuantityPrecision struct {
	Asset    string `json:"asset"`
	Fraction int64  `json:"fraction"`
}

// ConvertQuoteResponse represents a response quote for the requested token pairs
type ConvertQuoteResponse struct {
	QuoteID        string       `json:"quoteId"`
	Ratio          string       `json:"ratio"`
	InverseRatio   types.Number `json:"inverseRatio"`
	ValidTimestamp types.Time   `json:"validTimestamp"`
	ToAmount       types.Number `json:"toAmount"`
	FromAmount     types.Number `json:"fromAmount"`
}

// QuoteOrderStatus represent a response of accepting a quote.
type QuoteOrderStatus struct {
	OrderID     string     `json:"orderId"`
	CreateTime  types.Time `json:"createTime"`
	OrderStatus string     `json:"orderStatus"` // PROCESS/ACCEPT_SUCCESS/SUCCESS/FAIL
}

// ConvertOrderStatus represents a convert order status.
type ConvertOrderStatus struct {
	OrderID      int64        `json:"orderId"`
	OrderStatus  string       `json:"orderStatus"`
	FromAsset    string       `json:"fromAsset"`
	ToAsset      string       `json:"toAsset"`
	FromAmount   types.Number `json:"fromAmount"`
	ToAmount     types.Number `json:"toAmount"`
	Ratio        types.Number `json:"ratio"`
	InverseRatio types.Number `json:"inverseRatio"`
	CreateTime   types.Time   `json:"createTime"`
}

// ConvertPlaceLimitOrderParam represents a convert place limit order parameters.
type ConvertPlaceLimitOrderParam struct {
	BaseAsset   currency.Code `json:"baseAsset"` // base asset (use the response fromIsBase from GET /sapi/v1/convert/exchangeInfo api to check which one is baseAsset )
	QuoteAsset  currency.Code `json:"quoteAsset"`
	LimitPrice  float64       `json:"limitPrice"`
	BaseAmount  float64       `json:"baseAmount,omitempty"`  // Base asset amount. (One of baseAmount or quoteAmount is required)
	QuoteAmount float64       `json:"quoteAmount,omitempty"` // Quote asset amount. (One of baseAmount or quoteAmount is required)
	Side        string        `json:"side"`                  // BUY or SELL
	WalletType  string        `json:"walletType,omitempty"`  // SPOT or FUNDING or SPOT_FUNDING. It is to use which type of assets. Default is SPOT.
	ExpiredType string        `json:"expiredType"`           // 1_D, 3_D, 7_D, 30_D (D means day)
}

// OrderStatusResponse represents a convert limit order response.
type OrderStatusResponse struct {
	OrderID int64  `json:"orderId"`
	Status  string `json:"status"`
}

// LimitOrderHistory represents a limit order details.
type LimitOrderHistory struct {
	List []LimitOrderDetail `json:"list"`
}

// LimitOrderDetail represents a limit order detail information.
type LimitOrderDetail struct {
	QuoteID          string       `json:"quoteId"`
	OrderID          int64        `json:"orderId"`
	OrderStatus      string       `json:"orderStatus"`
	FromAsset        string       `json:"fromAsset"`
	FromAmount       types.Number `json:"fromAmount"`
	ToAsset          string       `json:"toAsset"`
	ToAmount         types.Number `json:"toAmount"`
	Ratio            types.Number `json:"ratio"`
	InverseRatio     types.Number `json:"inverseRatio"`
	CreateTime       types.Time   `json:"createTime"`
	ExpiredTimestamp types.Time   `json:"expiredTimestamp"`
}

// ConvertTradeHistory represents a response for convert trade history
type ConvertTradeHistory struct {
	List      []LimitOrderDetail `json:"list"`
	StartTime types.Time         `json:"startTime"`
	EndTime   types.Time         `json:"endTime"`
	Limit     int64              `json:"limit"`
	MoreData  bool               `json:"moreData"`
}

// RebateHistory represents a rebate history response
type RebateHistory struct {
	Status string `json:"status"`
	Type   string `json:"type"`
	Code   string `json:"code"`
	Data   struct {
		Page         int64 `json:"page"`
		TotalRecords int64 `json:"totalRecords"`
		TotalPageNum int64 `json:"totalPageNum"`
		Data         []struct {
			Asset      string       `json:"asset"`
			Type       int64        `json:"type"`
			Amount     types.Number `json:"amount"`
			UpdateTime types.Time   `json:"updateTime"`
		} `json:"data"`
	} `json:"data"`
}

// NFTTransactionHistory represents an NFT transaction history
type NFTTransactionHistory struct {
	Total int64 `json:"total"` // total records
	List  []struct {
		OrderNo string `json:"orderNo"` // 0: purchase order, 1: sell order, 2: royalty income, 3: primary market order, 4: mint fee
		Tokens  []struct {
			Network         string `json:"network"`         // NFT Network
			TokenID         string `json:"tokenId"`         // NFT Token ID
			ContractAddress string `json:"contractAddress"` // NFT Contract Address
		} `json:"tokens"`
		TradeTime     types.Time   `json:"tradeTime"`
		TradeAmount   types.Number `json:"tradeAmount"`
		TradeCurrency string       `json:"tradeCurrency"`
	} `json:"list"`
}

// NFTDepositHistory represents an NFT deposit history
type NFTDepositHistory struct {
	Total int64 `json:"total"`
	List  []struct {
		Network         string     `json:"network"`
		TransactionID   any        `json:"txID"`
		ContractAdrress string     `json:"contractAdrress"`
		TokenID         string     `json:"tokenId"`
		Timestamp       types.Time `json:"timestamp"`
	} `json:"list"`
}

// NFTWithdrawalHistory represents an NFT withdrawal history
type NFTWithdrawalHistory struct {
	Total int64 `json:"total"`
	List  []struct {
		Network         string     `json:"network"`
		TransactionID   string     `json:"txID"`
		ContractAdrress string     `json:"contractAdrress"`
		TokenID         string     `json:"tokenId"`
		Timestamp       types.Time `json:"timestamp"`
		Fee             float64    `json:"fee"`
		FeeAsset        string     `json:"feeAsset"`
	} `json:"list"`
}

// NFTAssets represents NFT assets list
type NFTAssets struct {
	Total int64 `json:"total"`
	List  []struct {
		Network         string `json:"network"`
		ContractAddress string `json:"contractAddress"`
		TokenID         string `json:"tokenId"`
	} `json:"list"`
}

// GiftCard represents a single-token gift card.
type GiftCard struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    struct {
		ReferenceNo string     `json:"referenceNo"`
		Code        string     `json:"code"`
		ExpiredTime types.Time `json:"expiredTime"`
	} `json:"data"`
	Success bool `json:"success"`
}

// DualTokenGiftCard represents a response for creating a dual token gift card.
type DualTokenGiftCard struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    struct {
		ReferenceNo string     `json:"referenceNo"`
		Code        string     `json:"code"`
		ExpiredTime types.Time `json:"expiredTime"`
	} `json:"data"`
	Success bool `json:"success"`
}

// RedeemBinanceGiftCard represents a binance gift card redemption response.
type RedeemBinanceGiftCard struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    struct {
		ReferenceNo string       `json:"referenceNo"`
		IdentityNo  string       `json:"identityNo"`
		Token       string       `json:"token"`
		Amount      types.Number `json:"amount"`
	} `json:"data"`
	Success bool `json:"success"`
}

// GiftCardVerificationResponse represents a Binance Gift Card verification response.
type GiftCardVerificationResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Valid  bool         `json:"valid"`
		Token  string       `json:"token"`
		Amount types.Number `json:"amount"`
	} `json:"data"`
	Success bool `json:"success"`
}

// RSAPublicKeyResponse represents an RSA public key response.
type RSAPublicKeyResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data"`
	Success bool   `json:"success"`
}

// TokenLimitInfo represents a token info
type TokenLimitInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    []struct {
		Coin    string       `json:"coin"`
		FromMin types.Number `json:"fromMin"`
		FromMax types.Number `json:"fromMax"`
	} `json:"data"`
	Success bool `json:"success"`
}

// VIPLoanRepaymentHistoryResponse represents a VIP loan repayment history response.
type VIPLoanRepaymentHistoryResponse struct {
	Rows []struct {
		LoanCoin       string       `json:"loanCoin"`
		RepayAmount    types.Number `json:"repayAmount"`
		CollateralCoin string       `json:"collateralCoin"`
		RepayStatus    string       `json:"repayStatus"`
		LoanDate       string       `json:"loanDate"`
		RepayTime      types.Time   `json:"repayTime"`
		OrderID        string       `json:"orderId"`
	} `json:"rows"`
	Total int64 `json:"total"`
}

// LoanRenewResponse represents loan renew
type LoanRenewResponse struct {
	LoanAccountID       string       `json:"loanAccountId"` // loan receiving account
	LoanCoin            string       `json:"loanCoin"`
	LoanAmount          types.Number `json:"loanAmount"`
	CollateralAccountID string       `json:"collateralAccountId"`
	CollateralCoin      string       `json:"collateralCoin"`
	LoanTerm            string       `json:"loanTerm"`
}

// LockedValueVIPCollateralAccount represents a collateral account locked response.
type LockedValueVIPCollateralAccount struct {
	Rows []struct {
		CollateralAccountID string `json:"collateralAccountId"`
		CollateralCoin      string `json:"collateralCoin"`
	} `json:"rows"`
	Total int64 `json:"total"`
}

// VIPLoanBorrow represents a VIP loan borrow detail.
type VIPLoanBorrow struct {
	LoanAccountID       string       `json:"loanAccountId"`
	RequestID           string       `json:"requestId"`
	LoanCoin            string       `json:"loanCoin"`
	IsFlexibleRate      string       `json:"isFlexibleRate"`
	LoanAmount          types.Number `json:"loanAmount"`
	CollateralAccountID string       `json:"collateralAccountId"`
	CollateralCoin      string       `json:"collateralCoin"`
	LoanTerm            string       `json:"loanTerm,omitempty"`
}

// VIPLoanableAssetsData represents a list of loanable assets for VIP account
type VIPLoanableAssetsData struct {
	Rows []struct {
		LoanCoin                   string       `json:"loanCoin"`
		FlexibleHourlyInterestRate types.Number `json:"_flexibleHourlyInterestRate"`
		FlexibleYearlyInterestRate types.Number `json:"_flexibleYearlyInterestRate"`
		Three0DDailyInterestRate   types.Number `json:"_30dDailyInterestRate"`
		Three0DYearlyInterestRate  types.Number `json:"_30dYearlyInterestRate"`
		Six0DDailyInterestRate     types.Number `json:"_60dDailyInterestRate"`
		Six0DYearlyInterestRate    types.Number `json:"_60dYearlyInterestRate"`
		MinLimit                   types.Number `json:"minLimit"`
		MaxLimit                   types.Number `json:"maxLimit"`
		VipLevel                   int64        `json:"vipLevel"`
	} `json:"rows"`
	Total int64 `json:"total"`
}

// VIPCollateralAssetData represents a VIP collateral asset data.
type VIPCollateralAssetData struct {
	Rows []struct {
		CollateralCoin         string `json:"collateralCoin"`
		OneStCollateralRatio   string `json:"_1stCollateralRatio"`
		OneStCollateralRange   string `json:"_1stCollateralRange"`
		TwoNdCollateralRatio   string `json:"_2ndCollateralRatio"`
		TwoNdCollateralRange   string `json:"_2ndCollateralRange"`
		ThreeRdCollateralRatio string `json:"_3rdCollateralRatio"`
		ThreeRdCollateralRange string `json:"_3rdCollateralRange"`
		FourThCollateralRatio  string `json:"_4thCollateralRatio"`
		FourThCollateralRange  string `json:"_4thCollateralRange"`
	} `json:"rows"`
	Total int64 `json:"total"`
}

// LoanApplicationStatus represents a loan application status response.
type LoanApplicationStatus struct {
	Rows []struct {
		LoanAccountID       string       `json:"loanAccountId"`
		OrderID             string       `json:"orderId"`
		RequestID           string       `json:"requestId"`
		LoanCoin            string       `json:"loanCoin"`
		LoanAmount          types.Number `json:"loanAmount"`
		CollateralAccountID string       `json:"collateralAccountId"`
		CollateralCoin      string       `json:"collateralCoin"`
		LoanTerm            string       `json:"loanTerm"`
		Status              string       `json:"status"`
		LoanDate            string       `json:"loanDate"`
	} `json:"rows"`
	Total int64 `json:"total"`
}

// BorrowInterestRate represents a borrow interest rate response.
type BorrowInterestRate struct {
	Asset                      string       `json:"asset"`
	FlexibleDailyInterestRate  types.Number `json:"flexibleDailyInterestRate"`
	FlexibleYearlyInterestRate types.Number `json:"flexibleYearlyInterestRate"`
	Time                       types.Time   `json:"time"`
}

// ListenKeyResponse represents a listen-key response instance.
type ListenKeyResponse struct {
	ListenKey string `json:"listenKey"`
}

// ErrResponse holds error response information.
type ErrResponse struct {
	Code    types.Number `json:"code"`
	Message string       `json:"msg"`
}
