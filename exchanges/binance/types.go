package binance

import (
	"bytes"
	"errors"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/types"
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
)

var (
	errAPIResponse                 = errors.New("API error response")
	errAlgoTypeRequired            = errors.New("algo type is required")
	errLeverageRequired            = errors.New("leverage is required")
	errPriceMatchWithPrice         = errors.New("priceMatch cannot be sent with price")
	errLoanTermMustBeSet           = errors.New("loan term must be set")
	errValidEmailRequired          = errors.New("valid email address is required")
	errPageNumberRequired          = errors.New("page number is required")
	errLimitNumberRequired         = errors.New("invalid limit")
	errEmptySubAccountAPIKey       = errors.New("invalid sub-account API key")
	errInvalidFuturesType          = errors.New("invalid futures types")
	errInvalidAccountType          = errors.New("invalid account type specified")
	errProductIDRequired           = errors.New("product ID is required")
	errProjectIDRequired           = errors.New("project ID is required")
	errPlanIDRequired              = errors.New("plan ID  is required")
	errIndexIDIsRequired           = errors.New("index ID is required")
	errTransferAlgorithmRequired   = errors.New("transfer algorithm is required")
	errUsernameRequired            = errors.New("user name is required")
	errTransferTypeRequired        = errors.New("transfer type is required")
	errNameRequired                = errors.New("name is required")
	errPositionIDRequired          = errors.New("position ID is required")
	errRedemptionAccountRequired   = errors.New("redemption account not specified")
	errOptionTypeRequired          = errors.New("optionType is required")
	errPageSizeRequired            = errors.New("page size is required")
	errPortfolioDetailRequired     = errors.New("portfolio detail is required")
	errPlanTypeRequired            = errors.New("planType is required")
	errTransactionIDRequired       = errors.New("transaction ID is required")
	errQuestionnaireRequired       = errors.New("questionnaire payload missing")
	errRequestIDRequired           = errors.New("request ID is required")
	errStartTimeRequired           = errors.New("start time is required")
	errStrategyTypeRequired        = errors.New("strategy type is required")
	errReferenceNumberRequired     = errors.New("reference number is required")
	errExpiredTypeRequired         = errors.New("expiredType is required")
	errQuoteIDRequired             = errors.New("quote ID is required")
	errAccountIDRequired           = errors.New("account ID is required")
	errAccountRequired             = errors.New("account information required")
	errConfigIDRequired            = errors.New("config ID is required")
	errLendingTypeRequired         = errors.New("lending type is required")
	errEmptyCurrencyCodes          = errors.New("assetNames are required")
	errSourceTypeRequired          = errors.New("source type required")
	errInvalidPercentageAmount     = errors.New("invalid percentage amount")
	errHashRateRequired            = errors.New("hash rate is required")
	errCodeRequired                = errors.New("code is required")
	errAddressRequired             = errors.New("address is required")
	errInvalidSubscriptionCycle    = errors.New("invalid subscription cycle")
	errDurationRequired            = errors.New("duration is required")
	errInvalidTransactionType      = errors.New("invalid transaction type")
	errPossibleValuesRequired      = errors.New("urgency field is required")
	errPlanStatusRequired          = errors.New("plan status is required")
	errUsageTypeRequired           = errors.New("usage type is required")
	errDownloadIDRequired          = errors.New("download id is required")
	errMarginChangeTypeInvalid     = errors.New("invalid margin changeType")
	errCancelReplaceModeRequired   = errors.New("cancelReplaceMode is required")
	errExpirationTimeRequired      = errors.New("expiration time is required")
	errSubAccountIDMissing         = errors.New("sub-account id is missing")
	errSubAccountStatusMissing     = errors.New("sub-account status missing")
	errCommissionValueRequired     = errors.New("commission value is required")
	errRewardTypeMissing           = errors.New("reward type is required")
	errInvalidBrokerID             = errors.New("missing brokerID")
	errBatchCancelRequiresSamePair = errors.New("batch cancellation requires a single asset type and pair")
)

// objectOrArray decodes a response that is a single object when one symbol is requested and an array otherwise
type objectOrArray[T any] []T

// UnmarshalJSON decodes a JSON object into a one-element slice and a JSON array into a slice
func (o *objectOrArray[T]) UnmarshalJSON(data []byte) error {
	if trimmed := bytes.TrimLeft(data, " \t\r\n"); len(trimmed) > 0 && trimmed[0] == '{' {
		var single T
		if err := json.Unmarshal(trimmed, &single); err != nil {
			return err
		}
		*o = objectOrArray[T]{single}
		return nil
	}
	return json.Unmarshal(data, (*[]T)(o))
}

// ExchangeInfoRequest holds the optional filters of an exchange information request. Symbols cannot be combined with
// Permissions or SymbolStatus
type ExchangeInfoRequest struct {
	Symbols currency.Pairs
	// Permissions defaults to SPOT, MARGIN and LEVERAGED; symbols with any other permission are listed only when it is
	// requested
	Permissions []string
	// ShowPermissionSets false leaves each symbol's permission sets empty; the endpoint defaults it to true
	ShowPermissionSets *bool
	SymbolStatus       string
}

// ExchangeInfoResponse holds the exchange trading rules and symbol information
type ExchangeInfoResponse struct {
	Timezone        string           `json:"timezone"`
	ServerTime      types.Time       `json:"serverTime"`
	RateLimits      []RateLimitItem  `json:"rateLimits"`
	ExchangeFilters []ExchangeFilter `json:"exchangeFilters"`
	Symbols         []SymbolInfo     `json:"symbols"`
	// SORs is sent only when smart order routing (SOR) is available
	SORs []SORInfo `json:"sors"`
}

// RateLimitItem holds a rate limit. Count, the usage counted against it, is sent by the unfilled order count query
// and the WebSocket API but not by the exchange information
type RateLimitItem struct {
	RateLimitType  string `json:"rateLimitType"`
	Interval       string `json:"interval"`
	IntervalNumber uint64 `json:"intervalNum"`
	Limit          uint64 `json:"limit"`
	Count          uint64 `json:"count"`
}

// ExchangeFilter holds an exchange filter, whose type determines which limit it sets
type ExchangeFilter struct {
	FilterType             string `json:"filterType"`
	MaxNumberOrders        uint64 `json:"maxNumOrders"`
	MaxNumberAlgoOrders    uint64 `json:"maxNumAlgoOrders"`
	MaxNumberIcebergOrders uint64 `json:"maxNumIcebergOrders"`
	MaxNumberOrderLists    uint64 `json:"maxNumOrderLists"`
}

// SymbolInfo holds a symbol's trading rules and filters
type SymbolInfo struct {
	Symbol             string        `json:"symbol"`
	Status             string        `json:"status"`
	BaseAsset          currency.Code `json:"baseAsset"`
	BaseAssetPrecision uint64        `json:"baseAssetPrecision"`
	QuoteAsset         currency.Code `json:"quoteAsset"`
	// QuotePrecision is to be removed in a future API version in favour of QuoteAssetPrecision
	QuotePrecision                  uint64         `json:"quotePrecision"`
	QuoteAssetPrecision             uint64         `json:"quoteAssetPrecision"`
	BaseCommissionPrecision         uint64         `json:"baseCommissionPrecision"`
	QuoteCommissionPrecision        uint64         `json:"quoteCommissionPrecision"`
	OrderTypes                      []string       `json:"orderTypes"`
	IcebergAllowed                  bool           `json:"icebergAllowed"`
	OCOAllowed                      bool           `json:"ocoAllowed"`
	OTOAllowed                      bool           `json:"otoAllowed"`
	OPOAllowed                      bool           `json:"opoAllowed"`
	QuoteOrderQuantityMarketAllowed bool           `json:"quoteOrderQtyMarketAllowed"`
	AllowTrailingStop               bool           `json:"allowTrailingStop"`
	CancelReplaceAllowed            bool           `json:"cancelReplaceAllowed"`
	AmendAllowed                    bool           `json:"amendAllowed"`
	PegInstructionsAllowed          bool           `json:"pegInstructionsAllowed"`
	IsSpotTradingAllowed            bool           `json:"isSpotTradingAllowed"`
	IsMarginTradingAllowed          bool           `json:"isMarginTradingAllowed"`
	Filters                         []SymbolFilter `json:"filters"`
	Permissions                     []string       `json:"permissions"`
	PermissionSets                  [][]string     `json:"permissionSets"`
	DefaultSelfTradePreventionMode  string         `json:"defaultSelfTradePreventionMode"`
	AllowedSelfTradePreventionModes []string       `json:"allowedSelfTradePreventionModes"`
}

// SymbolFilter holds a symbol filter, whose type determines which of the fields it sets
type SymbolFilter struct {
	FilterType string `json:"filterType"`
	// PRICE_FILTER
	MinPrice types.Number `json:"minPrice"`
	MaxPrice types.Number `json:"maxPrice"`
	TickSize types.Number `json:"tickSize"`
	// PERCENT_PRICE
	MultiplierUp   types.Number `json:"multiplierUp"`
	MultiplierDown types.Number `json:"multiplierDown"`
	// AveragePriceMinutes is the window of the average price the percent price and notional filters use; zero means the
	// last price
	AveragePriceMinutes uint64 `json:"avgPriceMins"`
	// PERCENT_PRICE_BY_SIDE
	BidMultiplierUp   types.Number `json:"bidMultiplierUp"`
	BidMultiplierDown types.Number `json:"bidMultiplierDown"`
	AskMultiplierUp   types.Number `json:"askMultiplierUp"`
	AskMultiplierDown types.Number `json:"askMultiplierDown"`
	// LOT_SIZE and MARKET_LOT_SIZE
	MinQuantity types.Number `json:"minQty"`
	MaxQuantity types.Number `json:"maxQty"`
	StepSize    types.Number `json:"stepSize"`
	// MIN_NOTIONAL and NOTIONAL
	MinNotional      types.Number `json:"minNotional"`
	ApplyToMarket    bool         `json:"applyToMarket"`
	ApplyMinToMarket bool         `json:"applyMinToMarket"`
	MaxNotional      types.Number `json:"maxNotional"`
	ApplyMaxToMarket bool         `json:"applyMaxToMarket"`
	// ICEBERG_PARTS
	Limit uint64 `json:"limit"`
	// MAX_NUM_ORDERS, MAX_NUM_ALGO_ORDERS, MAX_NUM_ICEBERG_ORDERS, MAX_NUM_ORDER_LISTS and MAX_NUM_ORDER_AMENDS
	MaxNumberOrders        uint64 `json:"maxNumOrders"`
	MaxNumberAlgoOrders    uint64 `json:"maxNumAlgoOrders"`
	MaxNumberIcebergOrders uint64 `json:"maxNumIcebergOrders"`
	MaxNumberOrderLists    uint64 `json:"maxNumOrderLists"`
	MaxNumberOrderAmends   uint64 `json:"maxNumOrderAmends"`
	// MAX_POSITION
	MaxPosition types.Number `json:"maxPosition"`
	// TRAILING_DELTA, in basis points
	MinTrailingAboveDelta uint64 `json:"minTrailingAboveDelta"`
	MaxTrailingAboveDelta uint64 `json:"maxTrailingAboveDelta"`
	MinTrailingBelowDelta uint64 `json:"minTrailingBelowDelta"`
	MaxTrailingBelowDelta uint64 `json:"maxTrailingBelowDelta"`
	// T_PLUS_SELL
	EndTime types.Time `json:"endTime"`
}

// SORInfo holds the symbols smart order routing (SOR) routes a base asset's orders between
type SORInfo struct {
	BaseAsset currency.Code `json:"baseAsset"`
	Symbols   []string      `json:"symbols"`
}

// ServerTimeResponse holds the server time
type ServerTimeResponse struct {
	ServerTime types.Time `json:"serverTime"`
}

// AggregatedTradeRequest holds the parameters of an aggregate trades request. FromID cannot be combined with
// StartTime or EndTime, and a Limit above 1000 is collected over several requests, which needs FromID or StartTime
type AggregatedTradeRequest struct {
	Symbol    currency.Pair
	FromID    uint64
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
}

// AggregatedTrade holds an aggregate trade: the fills of one taker order at one price
type AggregatedTrade struct {
	AggregateTradeID uint64       `json:"a"`
	Price            types.Number `json:"p"`
	Quantity         types.Number `json:"q"`
	FirstTradeID     uint64       `json:"f"`
	LastTradeID      uint64       `json:"l"`
	Timestamp        types.Time   `json:"T"`
	IsBuyerMaker     bool         `json:"m"`
	IsBestMatch      bool         `json:"M"`
}

// AveragePriceResponse holds a symbol's current average price
type AveragePriceResponse struct {
	Minutes   uint64       `json:"mins"` // Interval the price is averaged over
	Price     types.Number `json:"price"`
	CloseTime types.Time   `json:"closeTime"` // Time of the last trade
}

// OrderBookRequest holds the parameters of an order book request
type OrderBookRequest struct {
	Symbol currency.Pair
	// Limit is the number of levels on each side, 100 when zero and at most 5000
	Limit uint64
	// SymbolStatus rejects the request when the symbol's trading status differs
	SymbolStatus string
}

// OrderBookResponse holds an order book
type OrderBookResponse struct {
	LastUpdateID int64                            `json:"lastUpdateId"`
	Bids         orderbook.LevelsArrayPriceAmount `json:"bids"`
	Asks         orderbook.LevelsArrayPriceAmount `json:"asks"`
}

// RecentTradeRequest holds the parameters of a recent trades request
type RecentTradeRequest struct {
	Symbol currency.Pair
	Limit  uint64 // 500 when zero, at most 1000
}

// RecentTrade holds a trade of the recent or historical trades lists
type RecentTrade struct {
	ID            uint64       `json:"id"`
	Price         types.Number `json:"price"`
	Quantity      types.Number `json:"qty"`
	QuoteQuantity types.Number `json:"quoteQty"`
	Time          types.Time   `json:"time"`
	IsBuyerMaker  bool         `json:"isBuyerMaker"`
	IsBestMatch   bool         `json:"isBestMatch"`
}

// KlinesRequest holds the parameters of a klines or UIKlines request
type KlinesRequest struct {
	Symbol   currency.Pair
	Interval string
	Limit    uint64 // 500 when zero, at most 1000
	// StartTime and EndTime are always in UTC
	StartTime time.Time
	EndTime   time.Time
	// TimeZone sets the time zone the intervals are interpreted in, as hours and minutes or hours alone from -12:00 to
	// +14:00; UTC when empty
	TimeZone string
}

// CandleStick holds a kline, a candlestick bar
type CandleStick struct {
	OpenTime                 types.Time
	OpenPrice                types.Number
	HighPrice                types.Number
	LowPrice                 types.Number
	ClosePrice               types.Number
	Volume                   types.Number
	CloseTime                types.Time
	QuoteAssetVolume         types.Number
	NumberOfTrades           uint64
	TakerBuyBaseAssetVolume  types.Number
	TakerBuyQuoteAssetVolume types.Number
}

// UnmarshalJSON decodes a kline from its array form, whose twelfth element is unused
func (c *CandleStick) UnmarshalJSON(data []byte) error {
	target := [12]any{&c.OpenTime, &c.OpenPrice, &c.HighPrice, &c.LowPrice, &c.ClosePrice, &c.Volume, &c.CloseTime, &c.QuoteAssetVolume, &c.NumberOfTrades, &c.TakerBuyBaseAssetVolume, &c.TakerBuyQuoteAssetVolume, nil}
	return json.Unmarshal(data, &target)
}

// RollingWindowTickerRequest holds the parameters of a rolling window price change statistics request
type RollingWindowTickerRequest struct {
	Symbols currency.Pairs // At most 100
	// WindowSize is sent as whole minutes from 1 to 59, hours from 1 to 23 or days from 1 to 7; one day when zero
	WindowSize   time.Duration
	Type         string // FULL when empty, or MINI
	SymbolStatus string
}

// TickerStatistics holds a symbol's price change statistics over a trading day or a rolling window. The MINI type
// leaves out the price change fields
type TickerStatistics struct {
	Symbol               string       `json:"symbol"`
	PriceChange          types.Number `json:"priceChange"`
	PriceChangePercent   types.Number `json:"priceChangePercent"`
	WeightedAveragePrice types.Number `json:"weightedAvgPrice"`
	OpenPrice            types.Number `json:"openPrice"`
	HighPrice            types.Number `json:"highPrice"`
	LowPrice             types.Number `json:"lowPrice"`
	LastPrice            types.Number `json:"lastPrice"`
	Volume               types.Number `json:"volume"`
	QuoteVolume          types.Number `json:"quoteVolume"`
	OpenTime             types.Time   `json:"openTime"`
	CloseTime            types.Time   `json:"closeTime"`
	FirstID              int64        `json:"firstId"` // -1 when there were no trades
	LastID               int64        `json:"lastId"`  // -1 when there were no trades
	Count                uint64       `json:"count"`
}

// PriceChangeStatsRequest holds the parameters of a 24hr ticker price change statistics request
type PriceChangeStatsRequest struct {
	Symbols      currency.Pairs // Every symbol when empty
	Type         string         // FULL when empty, or MINI
	SymbolStatus string
}

// PriceChangeStats holds 24hr ticker price change statistics; the MINI type leaves out the price change, previous close,
// last quantity, bid and ask fields
type PriceChangeStats struct {
	Symbol               string       `json:"symbol"`
	PriceChange          types.Number `json:"priceChange"`
	PriceChangePercent   types.Number `json:"priceChangePercent"`
	WeightedAveragePrice types.Number `json:"weightedAvgPrice"`
	PreviousClosePrice   types.Number `json:"prevClosePrice"`
	LastPrice            types.Number `json:"lastPrice"`
	LastQuantity         types.Number `json:"lastQty"`
	BidPrice             types.Number `json:"bidPrice"`
	BidQuantity          types.Number `json:"bidQty"`
	AskPrice             types.Number `json:"askPrice"`
	AskQuantity          types.Number `json:"askQty"`
	OpenPrice            types.Number `json:"openPrice"`
	HighPrice            types.Number `json:"highPrice"`
	LowPrice             types.Number `json:"lowPrice"`
	Volume               types.Number `json:"volume"`
	QuoteVolume          types.Number `json:"quoteVolume"`
	OpenTime             types.Time   `json:"openTime"`
	CloseTime            types.Time   `json:"closeTime"`
	FirstID              int64        `json:"firstId"` // -1 when there were no trades
	LastID               int64        `json:"lastId"`  // -1 when there were no trades
	Count                uint64       `json:"count"`
}

// BookTicker holds a symbol's best bid and ask price and quantity
type BookTicker struct {
	Symbol      string       `json:"symbol"`
	BidPrice    types.Number `json:"bidPrice"`
	BidQuantity types.Number `json:"bidQty"`
	AskPrice    types.Number `json:"askPrice"`
	AskQuantity types.Number `json:"askQty"`
}

// SymbolPrice holds a symbol's latest price
type SymbolPrice struct {
	Symbol string       `json:"symbol"`
	Price  types.Number `json:"price"`
}

// TradingDayTickerRequest holds the parameters of a trading day ticker request
type TradingDayTickerRequest struct {
	Symbols currency.Pairs // At most 100
	// TimeZone sets the time zone the trading day starts in, as hours and minutes or hours alone; UTC when empty
	TimeZone     string
	Type         string // FULL when empty, or MINI
	SymbolStatus string
}

// OrderConditionalFields holds the order response fields that appear only when an order uses the feature behind them,
// such as an iceberg quantity, self-trade prevention, a stop price, a strategy, a trailing stop, smart order routing
// (SOR), a price peg or an expiry
type OrderConditionalFields struct {
	IcebergQuantity   types.Number `json:"icebergQty"`
	PreventedMatchID  uint64       `json:"preventedMatchId"`
	PreventedQuantity types.Number `json:"preventedQuantity"`
	StopPrice         types.Number `json:"stopPrice"`
	StrategyID        uint64       `json:"strategyId"`
	StrategyType      uint64       `json:"strategyType"`
	TrailingDelta     uint64       `json:"trailingDelta"`
	// TrailingTime is the Unix time in milliseconds the trailing stop started tracking the price, -1 until then
	TrailingTime   int64        `json:"trailingTime"`
	UsedSOR        bool         `json:"usedSor"`
	WorkingFloor   string       `json:"workingFloor"`
	PegPriceType   string       `json:"pegPriceType"`
	PegOffsetType  string       `json:"pegOffsetType"`
	PegOffsetValue uint64       `json:"pegOffsetValue"`
	PeggedPrice    types.Number `json:"peggedPrice"`
	ExpiryReason   string       `json:"expiryReason"`
}

// CancelOrderResponse holds a cancelled order
type CancelOrderResponse struct {
	Symbol                     string       `json:"symbol"`
	OriginalClientOrderID      string       `json:"origClientOrderId"`
	OrderID                    uint64       `json:"orderId"`
	OrderListID                int64        `json:"orderListId"` // -1 unless the order is part of an order list
	ClientOrderID              string       `json:"clientOrderId"`
	TransactTime               types.Time   `json:"transactTime"`
	Price                      types.Number `json:"price"`
	OriginalQuantity           types.Number `json:"origQty"`
	ExecutedQuantity           types.Number `json:"executedQty"`
	OriginalQuoteOrderQuantity types.Number `json:"origQuoteOrderQty"`
	CummulativeQuoteQuantity   types.Number `json:"cummulativeQuoteQty"`
	Status                     string       `json:"status"`
	TimeInForce                string       `json:"timeInForce"`
	Type                       string       `json:"type"`
	Side                       string       `json:"side"`
	SelfTradePreventionMode    string       `json:"selfTradePreventionMode"`
	OrderConditionalFields
}

// CancelledOpenOrder holds an entry of a cancel all open orders response: a cancelled order, or a cancelled order list
// with its list status, orders and order reports
type CancelledOpenOrder struct {
	CancelOrderResponse
	ContingencyType   string           `json:"contingencyType"`
	ListStatusType    string           `json:"listStatusType"`
	ListOrderStatus   string           `json:"listOrderStatus"`
	ListClientOrderID string           `json:"listClientOrderId"`
	TransactionTime   types.Time       `json:"transactionTime"`
	Orders            []OrderListOrder `json:"orders"`
	OrderReports      []OrderResponse  `json:"orderReports"`
}

// NewOrderRequest holds the parameters of a new order. Which quantity, price, stop and trailing parameters are
// mandatory depends on Type
type NewOrderRequest struct {
	Symbol             currency.Pair
	Side               string // BUY or SELL
	Type               string
	TimeInForce        string
	Quantity           float64
	QuoteOrderQuantity float64 // Quote asset amount a MARKET order spends or receives, instead of Quantity
	Price              float64
	NewClientOrderID   string
	StrategyID         uint64
	StrategyType       uint64 // At least 1000000; smaller values are reserved
	StopPrice          float64
	TrailingDelta      uint64 // In basis points
	IcebergQuantity    float64
	// NewOrderResponseType selects the ACK, RESULT or FULL response; MARKET and LIMIT orders default to FULL and the
	// others to ACK
	NewOrderResponseType    string
	SelfTradePreventionMode string
	PegPriceType            string // PRIMARY_PEG or MARKET_PEG, which makes Price optional
	PegOffsetValue          uint64 // Price level to peg to, at most 100
	PegOffsetType           string
}

// NewOrderResponse holds a new order in the ACK, RESULT or FULL format: ACK sets only the identifiers and transaction
// time, and FULL adds the fills to RESULT
type NewOrderResponse struct {
	Symbol                     string       `json:"symbol"`
	OrderID                    uint64       `json:"orderId"`
	OrderListID                int64        `json:"orderListId"` // -1 unless the order is part of an order list
	ClientOrderID              string       `json:"clientOrderId"`
	TransactTime               types.Time   `json:"transactTime"`
	Price                      types.Number `json:"price"`
	OriginalQuantity           types.Number `json:"origQty"`
	ExecutedQuantity           types.Number `json:"executedQty"`
	OriginalQuoteOrderQuantity types.Number `json:"origQuoteOrderQty"`
	CummulativeQuoteQuantity   types.Number `json:"cummulativeQuoteQty"`
	Status                     string       `json:"status"`
	TimeInForce                string       `json:"timeInForce"`
	Type                       string       `json:"type"`
	Side                       string       `json:"side"`
	// WorkingTime is the Unix time in milliseconds the order started working on the order book, -1 until then
	WorkingTime             int64       `json:"workingTime"`
	SelfTradePreventionMode string      `json:"selfTradePreventionMode"`
	Fills                   []OrderFill `json:"fills"`
	OrderConditionalFields
}

// OrderFill holds a fill of a new order
type OrderFill struct {
	Price           types.Number  `json:"price"`
	Quantity        types.Number  `json:"qty"`
	Commission      types.Number  `json:"commission"`
	CommissionAsset currency.Code `json:"commissionAsset"`
	TradeID         uint64        `json:"tradeId"`
}

// CancelOrderRequest holds the parameters of an order cancellation; OrderID or OriginalClientOrderID is required
type CancelOrderRequest struct {
	Symbol                currency.Pair
	OrderID               uint64
	OriginalClientOrderID string
	NewClientOrderID      string // Identifies the cancellation; generated when empty
	CancelRestrictions    string // ONLY_NEW or ONLY_PARTIALLY_FILLED cancels only orders with that status
}

// CancelOrderListRequest holds the parameters of an order list cancellation; OrderListID or ListClientOrderID is
// required
type CancelOrderListRequest struct {
	Symbol            currency.Pair
	OrderListID       uint64
	ListClientOrderID string
	NewClientOrderID  string // Identifies the cancellation; generated when empty
}

// OCOOrderResponse holds an order list; placements and cancellations add the order reports
type OCOOrderResponse struct {
	OrderListID       uint64           `json:"orderListId"`
	ContingencyType   string           `json:"contingencyType"`
	ListStatusType    string           `json:"listStatusType"`
	ListOrderStatus   string           `json:"listOrderStatus"`
	ListClientOrderID string           `json:"listClientOrderId"`
	TransactionTime   types.Time       `json:"transactionTime"`
	Symbol            string           `json:"symbol"`
	Orders            []OrderListOrder `json:"orders"`
	OrderReports      []OrderResponse  `json:"orderReports"`
}

// OrderListOrder identifies an order of an order list
type OrderListOrder struct {
	Symbol        string `json:"symbol"`
	OrderID       uint64 `json:"orderId"`
	ClientOrderID string `json:"clientOrderId"`
}

// OrderResponse holds an order report of an order list placement, in the placement's ACK, RESULT or FULL format, or of
// an order list cancellation, which alone sends OriginalClientOrderID
type OrderResponse struct {
	Symbol                     string            `json:"symbol"`
	OriginalClientOrderID      string            `json:"origClientOrderId"`
	OrderID                    uint64            `json:"orderId"`
	OrderListID                int64             `json:"orderListId"`
	ClientOrderID              string            `json:"clientOrderId"`
	TransactTime               types.Time        `json:"transactTime"`
	Price                      types.Number      `json:"price"`
	OriginalQuantity           types.Number      `json:"origQty"`
	ExecutedQuantity           types.Number      `json:"executedQty"`
	OriginalQuoteOrderQuantity types.Number      `json:"origQuoteOrderQty"`
	CummulativeQuoteQuantity   types.Number      `json:"cummulativeQuoteQty"`
	Status                     string            `json:"status"`
	TimeInForce                order.TimeInForce `json:"timeInForce"`
	Type                       string            `json:"type"`
	Side                       string            `json:"side"`
	// WorkingTime is the Unix time in milliseconds the order started working on the order book, -1 until then
	WorkingTime             int64       `json:"workingTime"`
	SelfTradePreventionMode string      `json:"selfTradePreventionMode"`
	Fills                   []OrderFill `json:"fills"`
	OrderConditionalFields
}

// CancelReplaceOrderRequest holds the parameters of a cancel-replace: the order to cancel, by CancelOrderID or
// CancelOriginalClientOrderID, and the new order, whose mandatory parameters depend on Type
type CancelReplaceOrderRequest struct {
	Symbol currency.Pair
	Side   string // BUY or SELL
	Type   string
	// CancelReplaceMode STOP_ON_FAILURE places the new order only if the cancellation succeeds, and ALLOW_FAILURE places
	// it either way
	CancelReplaceMode           string
	TimeInForce                 string
	Quantity                    float64
	QuoteOrderQuantity          float64
	Price                       float64
	CancelNewClientOrderID      string // Identifies the cancellation; generated when empty
	CancelOriginalClientOrderID string
	CancelOrderID               uint64
	NewClientOrderID            string
	StrategyID                  uint64
	StrategyType                uint64 // At least 1000000; smaller values are reserved
	StopPrice                   float64
	TrailingDelta               uint64 // In basis points
	IcebergQuantity             float64
	NewOrderResponseType        string // ACK, RESULT or FULL
	SelfTradePreventionMode     string
	CancelRestrictions          string // ONLY_NEW or ONLY_PARTIALLY_FILLED cancels only orders with that status
	// OrderRateLimitExceededMode DO_NOTHING, the default, cancels only within the unfilled order count limit, and
	// CANCEL_ONLY always cancels
	OrderRateLimitExceededMode string
	PegPriceType               string
	PegOffsetValue             uint64
	PegOffsetType              string
}

// CancelAndReplaceResponse holds the outcome of a cancel-replace. Binance replies with it when both halves succeed and
// puts it in the error's data when either fails, so a failed half holds an error code and message instead of an order
type CancelAndReplaceResponse struct {
	CancelResult     string                        `json:"cancelResult"`   // SUCCESS or FAILURE
	NewOrderResult   string                        `json:"newOrderResult"` // SUCCESS, FAILURE or NOT_ATTEMPTED
	CancelResponse   CancelReplaceCancelResponse   `json:"cancelResponse"`
	NewOrderResponse CancelReplaceNewOrderResponse `json:"newOrderResponse"`
}

// CancelReplaceCancelResponse holds the cancellation half of a cancel-replace: the cancelled order, or the error code
// and message of a failed cancellation
type CancelReplaceCancelResponse struct {
	CancelOrderResponse
	Code    int64  `json:"code"`
	Message string `json:"msg"`
}

// CancelReplaceNewOrderResponse holds the new order half of a cancel-replace: the placed order, or the error code and
// message of a failed placement
type CancelReplaceNewOrderResponse struct {
	NewOrderResponse
	Code    int64  `json:"code"`
	Message string `json:"msg"`
}

// OCOOrderListRequest holds the parameters of a one-cancels-the-other (OCO) order list
type OCOOrderListRequest struct {
	Symbol                  currency.Pair
	ListClientOrderID       string
	Side                    string  // BUY or SELL
	Quantity                float64 // Quantity of both orders
	Above                   OCOOrderListLeg
	Below                   OCOOrderListLeg
	NewOrderResponseType    string // ACK, RESULT or FULL
	SelfTradePreventionMode string
}

// OCOOrderListLeg holds the parameters of the above or below order of an OCO order list
type OCOOrderListLeg struct {
	Type            string
	ClientOrderID   string
	IcebergQuantity uint64  // A whole number, usable only with the GTC time in force
	Price           float64 // Limit price of a LIMIT_MAKER, STOP_LOSS_LIMIT or TAKE_PROFIT_LIMIT order
	StopPrice       float64
	TrailingDelta   uint64 // In basis points
	TimeInForce     string // Required by STOP_LOSS_LIMIT and TAKE_PROFIT_LIMIT orders
	StrategyID      uint64
	StrategyType    uint64 // At least 1000000; smaller values are reserved
	PegPriceType    string
	PegOffsetType   string
	PegOffsetValue  uint64
}

// OrderTestResponse holds the commission rates a test order would pay, which are sent only when they are requested
type OrderTestResponse struct {
	StandardCommissionForOrder OrderCommissionRates `json:"standardCommissionForOrder"`
	SpecialCommissionForOrder  OrderCommissionRates `json:"specialCommissionForOrder"`
	TaxCommissionForOrder      OrderCommissionRates `json:"taxCommissionForOrder"`
	Discount                   CommissionDiscount   `json:"discount"`
}

// OrderCommissionRates holds the maker and taker commission rates of an order
type OrderCommissionRates struct {
	Maker types.Number `json:"maker"`
	Taker types.Number `json:"taker"`
}

// CommissionDiscount holds the discount on standard commissions paid in the discount asset
type CommissionDiscount struct {
	EnabledForAccount bool          `json:"enabledForAccount"`
	EnabledForSymbol  bool          `json:"enabledForSymbol"`
	DiscountAsset     currency.Code `json:"discountAsset"`
	Discount          types.Number  `json:"discount"` // Rate the standard commission is reduced by
}

// SOROrderRequest holds the parameters of an order using smart order routing (SOR), which supports only LIMIT and
// MARKET orders without a quote order quantity
type SOROrderRequest struct {
	Symbol                  currency.Pair
	Side                    string // BUY or SELL
	Type                    string // LIMIT or MARKET
	TimeInForce             string
	Quantity                float64
	Price                   float64
	NewClientOrderID        string
	StrategyID              uint64
	StrategyType            uint64 // At least 1000000; smaller values are reserved
	IcebergQuantity         float64
	NewOrderResponseType    string // ACK, RESULT or FULL; FULL when empty
	SelfTradePreventionMode string
}

// SOROrderResponse holds an order placed using smart order routing (SOR)
type SOROrderResponse struct {
	Symbol                     string       `json:"symbol"`
	OrderID                    uint64       `json:"orderId"`
	OrderListID                int64        `json:"orderListId"`
	ClientOrderID              string       `json:"clientOrderId"`
	TransactTime               types.Time   `json:"transactTime"`
	Price                      types.Number `json:"price"`
	OriginalQuantity           types.Number `json:"origQty"`
	ExecutedQuantity           types.Number `json:"executedQty"`
	OriginalQuoteOrderQuantity types.Number `json:"origQuoteOrderQty"`
	CummulativeQuoteQuantity   types.Number `json:"cummulativeQuoteQty"`
	Status                     string       `json:"status"`
	TimeInForce                string       `json:"timeInForce"`
	Type                       string       `json:"type"`
	Side                       string       `json:"side"`
	// WorkingTime is the Unix time in milliseconds the order started working on the order book, -1 until then
	WorkingTime             int64     `json:"workingTime"`
	Fills                   []SORFill `json:"fills"`
	SelfTradePreventionMode string    `json:"selfTradePreventionMode"`
	OrderConditionalFields
}

// SORFill holds a fill of an order placed using smart order routing (SOR)
type SORFill struct {
	MatchType       string        `json:"matchType"`
	Price           types.Number  `json:"price"`
	Quantity        types.Number  `json:"qty"`
	Commission      types.Number  `json:"commission"`
	CommissionAsset currency.Code `json:"commissionAsset"`
	TradeID         int64         `json:"tradeId"` // -1 for a fill matched through a trade report
	AllocationID    uint64        `json:"allocId"`
}

// AccountCommissionResponse holds the account's commission rates on a symbol
type AccountCommissionResponse struct {
	Symbol             string             `json:"symbol"`
	StandardCommission CommissionRates    `json:"standardCommission"`
	SpecialCommission  CommissionRates    `json:"specialCommission"`
	TaxCommission      CommissionRates    `json:"taxCommission"`
	Discount           CommissionDiscount `json:"discount"`
}

// CommissionRates holds maker, taker, buyer and seller commission rates
type CommissionRates struct {
	Maker  types.Number `json:"maker"`
	Taker  types.Number `json:"taker"`
	Buyer  types.Number `json:"buyer"`
	Seller types.Number `json:"seller"`
}

// AllOrderListsRequest holds the optional filters of an all order lists request. FromID cannot be combined with
// StartTime or EndTime, which can be at most 24 hours apart
type AllOrderListsRequest struct {
	FromID    uint64
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64 // 500 when zero, at most 1000
}

// AllOrdersRequest holds the parameters of an all orders request. OrderID returns orders from that ID on, and
// StartTime and EndTime can be at most 24 hours apart
type AllOrdersRequest struct {
	Symbol    currency.Pair
	OrderID   uint64
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64 // 500 when zero, at most 1000
}

// TradeOrderResponse holds an order as queried
type TradeOrderResponse struct {
	Symbol                     string            `json:"symbol"`
	OrderID                    uint64            `json:"orderId"`
	OrderListID                int64             `json:"orderListId"` // -1 unless the order is part of an order list
	ClientOrderID              string            `json:"clientOrderId"`
	Price                      types.Number      `json:"price"`
	OriginalQuantity           types.Number      `json:"origQty"`
	ExecutedQuantity           types.Number      `json:"executedQty"`
	OriginalQuoteOrderQuantity types.Number      `json:"origQuoteOrderQty"`
	CummulativeQuoteQuantity   types.Number      `json:"cummulativeQuoteQty"` // Negative when unavailable for some historical orders
	Status                     string            `json:"status"`
	TimeInForce                order.TimeInForce `json:"timeInForce"`
	Type                       string            `json:"type"`
	Side                       string            `json:"side"`
	Time                       types.Time        `json:"time"`
	UpdateTime                 types.Time        `json:"updateTime"`
	IsWorking                  bool              `json:"isWorking"`
	// WorkingTime is the Unix time in milliseconds the order started working on the order book, -1 until then
	WorkingTime             int64  `json:"workingTime"`
	SelfTradePreventionMode string `json:"selfTradePreventionMode"`
	OrderConditionalFields
}

// AccountResponse holds the account information. The commissions are in basis points and signed, since a liquidity
// provider's maker rebate is reported as a negative commission
type AccountResponse struct {
	MakerCommission            int64           `json:"makerCommission"`
	TakerCommission            int64           `json:"takerCommission"`
	BuyerCommission            int64           `json:"buyerCommission"`
	SellerCommission           int64           `json:"sellerCommission"`
	CommissionRates            CommissionRates `json:"commissionRates"`
	CanTrade                   bool            `json:"canTrade"`
	CanWithdraw                bool            `json:"canWithdraw"`
	CanDeposit                 bool            `json:"canDeposit"`
	Brokered                   bool            `json:"brokered"`
	RequireSelfTradePrevention bool            `json:"requireSelfTradePrevention"`
	PreventSOR                 bool            `json:"preventSor"`
	UpdateTime                 types.Time      `json:"updateTime"`
	AccountType                string          `json:"accountType"`
	Balances                   []Balance       `json:"balances"`
	Permissions                []string        `json:"permissions"`
	UID                        uint64          `json:"uid"`
}

// Balance holds an asset's free and locked balance
type Balance struct {
	Asset  currency.Code   `json:"asset"`
	Free   decimal.Decimal `json:"free"`
	Locked decimal.Decimal `json:"locked"`
}

// AllocationsRequest holds the parameters of an allocations request; StartTime and EndTime can be at most 24 hours
// apart
type AllocationsRequest struct {
	Symbol           currency.Pair
	StartTime        time.Time
	EndTime          time.Time
	FromAllocationID uint64
	Limit            uint64 // 500 when zero, at most 1000
	OrderID          uint64
}

// Allocation holds an allocation resulting from an order placed using smart order routing (SOR)
type Allocation struct {
	Symbol          string        `json:"symbol"`
	AllocationID    uint64        `json:"allocationId"`
	AllocationType  string        `json:"allocationType"`
	OrderID         uint64        `json:"orderId"`
	OrderListID     int64         `json:"orderListId"`
	Price           types.Number  `json:"price"`
	Quantity        types.Number  `json:"qty"`
	QuoteQuantity   types.Number  `json:"quoteQty"`
	Commission      types.Number  `json:"commission"`
	CommissionAsset currency.Code `json:"commissionAsset"`
	Time            types.Time    `json:"time"`
	IsBuyer         bool          `json:"isBuyer"`
	IsMaker         bool          `json:"isMaker"`
	IsAllocator     bool          `json:"isAllocator"`
}

// PreventedMatchesRequest holds the parameters of a prevented matches request, by PreventedMatchID or by OrderID,
// which FromPreventedMatchID and Limit page through
type PreventedMatchesRequest struct {
	Symbol               currency.Pair
	PreventedMatchID     uint64
	OrderID              uint64
	FromPreventedMatchID uint64
	Limit                uint64 // 500 when zero, at most 1000
}

// PreventedMatch holds a match that self-trade prevention (STP) prevented, expiring the orders involved
type PreventedMatch struct {
	Symbol                  string       `json:"symbol"`
	PreventedMatchID        uint64       `json:"preventedMatchId"`
	TakerOrderID            uint64       `json:"takerOrderId"`
	MakerSymbol             string       `json:"makerSymbol"`
	MakerOrderID            uint64       `json:"makerOrderId"`
	TradeGroupID            uint64       `json:"tradeGroupId"`
	SelfTradePreventionMode string       `json:"selfTradePreventionMode"`
	Price                   types.Number `json:"price"`
	MakerPreventedQuantity  types.Number `json:"makerPreventedQuantity"`
	TransactTime            types.Time   `json:"transactTime"`
}

// AccountTradeListRequest holds the parameters of an account trade list request; StartTime and EndTime can be at most
// 24 hours apart
type AccountTradeListRequest struct {
	Symbol    currency.Pair
	OrderID   uint64
	StartTime time.Time
	EndTime   time.Time
	FromID    uint64 // Trades from this trade ID on
	Limit     uint64 // 500 when zero, at most 1000
}

// AccountTrade holds a trade of the account
type AccountTrade struct {
	Symbol          string        `json:"symbol"`
	ID              uint64        `json:"id"`
	OrderID         uint64        `json:"orderId"`
	OrderListID     int64         `json:"orderListId"`
	Price           types.Number  `json:"price"`
	Quantity        types.Number  `json:"qty"`
	QuoteQuantity   types.Number  `json:"quoteQty"`
	Commission      types.Number  `json:"commission"`
	CommissionAsset currency.Code `json:"commissionAsset"`
	Time            types.Time    `json:"time"`
	IsBuyer         bool          `json:"isBuyer"`
	IsMaker         bool          `json:"isMaker"`
	IsBestMatch     bool          `json:"isBestMatch"`
}

// BNBBurnStatusResponse represents a response of spot trade and margin interest
type BNBBurnStatusResponse struct {
	SpotBNBBurn     bool `json:"spotBNBBurn"`
	InterestBNBBurn bool `json:"interestBNBBurn"`
}

// ErrResponse holds error response information
type ErrResponse struct {
	Code    types.Number    `json:"code"`
	Message string          `json:"msg"`
	Data    json.RawMessage `json:"data"` // Only some endpoints add details, such as the outcome of a cancel-replace
}
