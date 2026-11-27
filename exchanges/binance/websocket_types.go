package binance

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// klineInterval pairs a candle interval with its Binance wire representation.
type klineInterval struct {
	Interval kline.Interval
	String   string
}

// WebsocketPayloadRequest is a WebSocket Streams subscription request: SUBSCRIBE or UNSUBSCRIBE with the stream names
type WebsocketPayloadRequest struct {
	Method string   `json:"method"`
	Params []string `json:"params"`
	ID     string   `json:"id"`
}

// WebsocketServerShutdown is the serverShutdown event, sent on the stream and WebSocket API connections when the server
// is about to shut down
type WebsocketServerShutdown struct {
	EventType string     `json:"e"`
	EventTime types.Time `json:"E"`
}

// WebsocketPartialDepth is a Partial Book Depth Streams (<symbol>@depth<levels>) payload: the top levels of the book
type WebsocketPartialDepth struct {
	// LastUpdateID is signed like the order book's update IDs it is loaded into
	LastUpdateID int64                            `json:"lastUpdateId"`
	Bids         orderbook.LevelsArrayPriceAmount `json:"bids"`
	Asks         orderbook.LevelsArrayPriceAmount `json:"asks"`
}

// DiffDepthStream is a Diff. Depth Stream (<symbol>@depth) payload: the new quantity of each changed price level
type DiffDepthStream struct {
	EventType     string                           `json:"e"`
	EventTime     types.Time                       `json:"E"`
	Symbol        string                           `json:"s"`
	FirstUpdateID int64                            `json:"U"` // int64 to match the order book API it feeds
	FinalUpdateID int64                            `json:"u"` // int64 to match the order book API it feeds
	Bids          orderbook.LevelsArrayPriceAmount `json:"b"`
	Asks          orderbook.LevelsArrayPriceAmount `json:"a"`
}

// KlineStream is a Kline/Candlestick Streams (<symbol>@kline_<interval>) payload
type KlineStream struct {
	EventType string     `json:"e"`
	EventTime types.Time `json:"E"`
	Symbol    string     `json:"s"`
	Kline     SpotCandle `json:"k"`
}

// SpotCandle is the candle of a kline stream payload
type SpotCandle struct {
	StartTime                types.Time   `json:"t"`
	CloseTime                types.Time   `json:"T"`
	Symbol                   string       `json:"s"`
	Interval                 string       `json:"i"`
	FirstTradeID             int64        `json:"f"` // -1 when the candle has no trades
	LastTradeID              int64        `json:"L"` // -1 when the candle has no trades
	OpenPrice                types.Number `json:"o"`
	ClosePrice               types.Number `json:"c"`
	HighPrice                types.Number `json:"h"`
	LowPrice                 types.Number `json:"l"`
	BaseAssetVolume          types.Number `json:"v"`
	NumberOfTrades           uint64       `json:"n"`
	IsClosed                 bool         `json:"x"`
	QuoteAssetVolume         types.Number `json:"q"`
	TakerBuyBaseAssetVolume  types.Number `json:"V"`
	TakerBuyQuoteAssetVolume types.Number `json:"Q"`
	Ignore                   types.Number `json:"B"`
}

// TickerStream is an Individual Symbol Ticker Streams (<symbol>@ticker) payload: 24hr rolling window statistics
type TickerStream struct {
	EventType          string       `json:"e"`
	EventTime          types.Time   `json:"E"`
	Symbol             string       `json:"s"`
	PriceChange        types.Number `json:"p"`
	PriceChangePercent types.Number `json:"P"`
	// WeightedAveragePrice is the volume weighted average price of the window
	WeightedAveragePrice types.Number `json:"w"`
	// PreviousClosePrice is the price of the last trade before the window, which the REST ticker calls prevClosePrice
	PreviousClosePrice          types.Number `json:"x"`
	LastPrice                   types.Number `json:"c"`
	LastQuantity                types.Number `json:"Q"`
	BestBidPrice                types.Number `json:"b"`
	BestBidQuantity             types.Number `json:"B"`
	BestAskPrice                types.Number `json:"a"`
	BestAskQuantity             types.Number `json:"A"`
	OpenPrice                   types.Number `json:"o"`
	HighPrice                   types.Number `json:"h"`
	LowPrice                    types.Number `json:"l"`
	TotalTradedBaseAssetVolume  types.Number `json:"v"`
	TotalTradedQuoteAssetVolume types.Number `json:"q"`
	StatisticsOpenTime          types.Time   `json:"O"`
	StatisticsCloseTime         types.Time   `json:"C"`
	FirstTradeID                int64        `json:"F"` // -1 when the window has no trades
	LastTradeID                 int64        `json:"L"` // -1 when the window has no trades
	TotalNumberOfTrades         uint64       `json:"n"`
}

// TradeStream is a Trade Streams (<symbol>@trade) payload
type TradeStream struct {
	EventType    string       `json:"e"`
	EventTime    types.Time   `json:"E"`
	Symbol       string       `json:"s"`
	TradeID      uint64       `json:"t"`
	Price        types.Number `json:"p"`
	Quantity     types.Number `json:"q"`
	TradeTime    types.Time   `json:"T"`
	IsBuyerMaker bool         `json:"m"`
	Ignore       bool         `json:"M"`
}

// WebsocketAPIRequest is the envelope of a WebSocket API request
type WebsocketAPIRequest struct {
	ID     string         `json:"id"`
	Method string         `json:"method"`
	Params map[string]any `json:"params,omitempty"`
}

// WebsocketAPIResponse is the envelope of a WebSocket API response; Result is present when the request succeeded and
// Error when it failed
type WebsocketAPIResponse struct {
	ID         string                       `json:"id"`
	Status     uint64                       `json:"status"`
	Result     json.RawMessage              `json:"result"`
	Error      *WebsocketAPIError           `json:"error"`
	RateLimits []WebsocketRateLimitResponse `json:"rateLimits"`
}

// WebsocketAPIError is a failed WebSocket API request's error
type WebsocketAPIError struct {
	Code    int64  `json:"code"` // Binance's error codes are negative
	Message string `json:"msg"`
	// Data holds method specific details, such as retryAfter when rate limited or the outcome of each half of a
	// cancel-replace
	Data json.RawMessage `json:"data"`
}

// WebsocketRateLimitResponse is a rate limit's current usage
type WebsocketRateLimitResponse struct {
	RateLimitType  string `json:"rateLimitType"`
	Interval       string `json:"interval"`
	IntervalNumber uint64 `json:"intervalNum"`
	Limit          uint64 `json:"limit"`
	Count          uint64 `json:"count"`
}

// WebsocketAveragePriceResponse is the Current average price (avgPrice) result
type WebsocketAveragePriceResponse struct {
	IntervalInMinutes uint64       `json:"mins"`
	Price             types.Number `json:"price"`
	// CloseTime is the time of the last trade
	CloseTime types.Time `json:"closeTime"`
}

// WebsocketKlinesRequest holds the parameters of the Klines (klines) and UI Klines (uiKlines) requests
type WebsocketKlinesRequest struct {
	Symbol    currency.Pair
	Interval  kline.Interval
	StartTime time.Time
	EndTime   time.Time
	// TimeZone is the offset klines are interpreted in, e.g. "-1:00", "05:45" or "8"; UTC when empty
	TimeZone string
	Limit    uint64
}

// WebsocketRollingWindowTickerRequest holds the parameters of the Rolling window price change statistics (ticker)
// request
type WebsocketRollingWindowTickerRequest struct {
	// Symbols are sent as symbol when there is one and as symbols when there are several
	Symbols currency.Pairs
	// WindowSize is a whole number of minutes (1 to 59), hours (1 to 23) or days (1 to 7); 1 day when zero
	WindowSize time.Duration
	// Type is FULL (the default) or MINI
	Type TickerType
	// SymbolStatus filters the symbols by trading status: TRADING, HALT, BREAK or CANCEL_ONLY
	SymbolStatus SymbolStatus
}

// WebsocketTickerRequest holds the parameters of the 24hr ticker price change statistics (ticker.24hr) request
type WebsocketTickerRequest struct {
	// Symbols are sent as symbol when there is one and as symbols when there are several; every symbol when empty
	Symbols currency.Pairs
	// Type is FULL (the default) or MINI
	Type TickerType
	// SymbolStatus filters the symbols by trading status: TRADING, HALT, BREAK or CANCEL_ONLY
	SymbolStatus SymbolStatus
}

// WebsocketBookTickerResponse is a Symbol order book ticker (ticker.book) result entry
type WebsocketBookTickerResponse struct {
	Symbol      string       `json:"symbol"`
	BidPrice    types.Number `json:"bidPrice"`
	BidQuantity types.Number `json:"bidQty"`
	AskPrice    types.Number `json:"askPrice"`
	AskQuantity types.Number `json:"askQty"`
}

// WebsocketPriceTickerResponse is a Symbol price ticker (ticker.price) result entry
type WebsocketPriceTickerResponse struct {
	Symbol string       `json:"symbol"`
	Price  types.Number `json:"price"`
}

// WebsocketTradingDayTickerRequest holds the parameters of the Trading Day Ticker (ticker.tradingDay) request
type WebsocketTradingDayTickerRequest struct {
	// Symbols are sent as symbol when there is one and as symbols when there are several
	Symbols currency.Pairs
	// TimeZone is the offset the trading day starts in, e.g. "-1:00", "05:45" or "8"; UTC when empty
	TimeZone string
	// Type is FULL (the default) or MINI
	Type TickerType
	// SymbolStatus filters the symbols by trading status: TRADING, HALT, BREAK or CANCEL_ONLY
	SymbolStatus SymbolStatus
}

// WebsocketAggregatedTradesRequest holds the parameters of the Aggregate trades (trades.aggregate) request; FromID
// cannot be combined with StartTime and EndTime
type WebsocketAggregatedTradesRequest struct {
	Symbol    currency.Pair
	FromID    uint64
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
}

// WebsocketSessionStatusResponse is the Query session status (session.status) and Log out of the session
// (session.logout) result
type WebsocketSessionStatusResponse struct {
	// APIKey is empty when the session is not authenticated
	APIKey string `json:"apiKey"`
	// AuthorisedSince is zero when the session is not authenticated
	AuthorisedSince  types.Time `json:"authorizedSince"`
	ConnectedSince   types.Time `json:"connectedSince"`
	ReturnRateLimits bool       `json:"returnRateLimits"`
	ServerTime       types.Time `json:"serverTime"`
	// UserDataStream reports whether the session has an active User Data Stream subscription; nil when Binance sends
	// null or leaves it out, so an unknown state is not mistaken for no subscription
	UserDataStream *bool `json:"userDataStream"`
}

// WebsocketOrderReport is an order's state as order cancellation results report it; the conditional fields appear only
// for orders they apply to
type WebsocketOrderReport struct {
	Symbol string `json:"symbol"`
	// OriginalClientOrderID is the client order ID the order had before it was cancelled
	OriginalClientOrderID      string       `json:"origClientOrderId"`
	OrderID                    uint64       `json:"orderId"`
	OrderListID                int64        `json:"orderListId"` // -1 unless the order is an order list leg
	ClientOrderID              string       `json:"clientOrderId"`
	TransactTime               types.Time   `json:"transactTime"`
	Price                      types.Number `json:"price"`
	OriginalQuantity           types.Number `json:"origQty"`
	ExecutedQuantity           types.Number `json:"executedQty"`
	OriginalQuoteOrderQuantity types.Number `json:"origQuoteOrderQty"`
	CumulativeQuoteQuantity    types.Number `json:"cummulativeQuoteQty"`
	Status                     string       `json:"status"`
	TimeInForce                TimeInForce  `json:"timeInForce"`
	Type                       OrderType    `json:"type"`
	Side                       Side         `json:"side"`
	StopPrice                  types.Number `json:"stopPrice"`
	TrailingDeltaInBasisPoints uint64       `json:"trailingDelta"`
	// TrailingTime is when a trailing stop order started tracking the price, in milliseconds; -1 before it activates
	TrailingTime            int64                   `json:"trailingTime"`
	IcebergQuantity         types.Number            `json:"icebergQty"`
	StrategyID              uint64                  `json:"strategyId"`
	StrategyType            uint64                  `json:"strategyType"`
	SelfTradePreventionMode SelfTradePreventionMode `json:"selfTradePreventionMode"`
	PreventedMatchID        uint64                  `json:"preventedMatchId"`
	PreventedQuantity       types.Number            `json:"preventedQuantity"`
	UsedSOR                 bool                    `json:"usedSor"`
	WorkingFloor            string                  `json:"workingFloor"`
	PegPriceType            PegPriceType            `json:"pegPriceType"`
	PegOffsetType           PegOffsetType           `json:"pegOffsetType"`
	PegOffsetValue          uint64                  `json:"pegOffsetValue"`
	PeggedPrice             types.Number            `json:"peggedPrice"`
	ExpiryReason            string                  `json:"expiryReason"`
}

// WebsocketOrderListOrder is an order list's leg
type WebsocketOrderListOrder struct {
	Symbol        string `json:"symbol"`
	OrderID       uint64 `json:"orderId"`
	ClientOrderID string `json:"clientOrderId"`
}

// WebsocketCancelOrderResponse is the Cancel order (order.cancel) and Cancel Order list (orderList.cancel) result and a
// Cancel open orders (openOrders.cancelAll) result entry. Cancelling an order list, or one of its legs, cancels the
// whole list, which is reported with the list fields and each leg's report instead of the order fields
type WebsocketCancelOrderResponse struct {
	WebsocketOrderReport
	ContingencyType   string                    `json:"contingencyType"`
	ListStatusType    string                    `json:"listStatusType"`
	ListOrderStatus   string                    `json:"listOrderStatus"`
	ListClientOrderID string                    `json:"listClientOrderId"`
	TransactionTime   types.Time                `json:"transactionTime"`
	Orders            []WebsocketOrderListOrder `json:"orders"`
	OrderReports      []WebsocketOrderReport    `json:"orderReports"`
}

// WebsocketCancelOrderRequest holds the parameters of the Cancel order (order.cancel) request; OrderID or
// OriginalClientOrderID identifies the order
type WebsocketCancelOrderRequest struct {
	Symbol                currency.Pair
	OrderID               uint64
	OriginalClientOrderID string
	// NewClientOrderID replaces the cancelled order's client order ID, freeing it for new orders
	NewClientOrderID string
	// CancelRestrictions limits the cancellation to orders with status NEW (ONLY_NEW) or PARTIALLY_FILLED
	// (ONLY_PARTIALLY_FILLED)
	CancelRestrictions CancelRestrictions
}

// WebsocketCancelReplaceOrderRequest holds the parameters of the Cancel and replace order (order.cancelReplace)
// request; CancelOrderID or CancelOriginalClientOrderID identifies the order to cancel
type WebsocketCancelReplaceOrderRequest struct {
	Symbol currency.Pair
	// CancelReplaceMode is STOP_ON_FAILURE or ALLOW_FAILURE
	CancelReplaceMode           CancelReplaceMode
	CancelOrderID               uint64
	CancelOriginalClientOrderID string
	CancelNewClientOrderID      string
	Side                        Side
	Type                        OrderType
	TimeInForce                 TimeInForce
	Price                       float64
	Quantity                    float64
	QuoteOrderQuantity          float64
	NewClientOrderID            string
	// NewOrderResponseType is ACK, RESULT or FULL
	NewOrderResponseType       NewOrderResponseType
	StopPrice                  float64
	TrailingDeltaInBasisPoints uint64
	IcebergQuantity            float64
	StrategyID                 uint64
	StrategyType               uint64
	SelfTradePreventionMode    SelfTradePreventionMode
	CancelRestrictions         CancelRestrictions
	// OrderRateLimitExceededMode is DO_NOTHING (the default) or CANCEL_ONLY
	OrderRateLimitExceededMode OrderRateLimitExceededMode
	PegPriceType               PegPriceType
	PegOffsetValue             uint64
	PegOffsetType              PegOffsetType
}

// WebsocketCancelReplaceOrderResponse is the Cancel and replace order (order.cancelReplace) result. Binance replies
// with it when both halves succeed and puts it in the error's data when either fails, so a failed half holds an error
// code and message instead of an order
type WebsocketCancelReplaceOrderResponse struct {
	// CancelResult is SUCCESS or FAILURE
	CancelResult string `json:"cancelResult"`
	// NewOrderResult is SUCCESS, FAILURE or NOT_ATTEMPTED
	NewOrderResult   string                                 `json:"newOrderResult"`
	CancelResponse   WebsocketCancelReplaceCancelResponse   `json:"cancelResponse"`
	NewOrderResponse WebsocketCancelReplaceNewOrderResponse `json:"newOrderResponse"`
}

// WebsocketCancelReplaceCancelResponse holds the cancellation half of a cancel and replace order: the cancelled order,
// or the error code and message of a failed cancellation
type WebsocketCancelReplaceCancelResponse struct {
	WebsocketCancelOrderResponse
	Code    int64  `json:"code"` // negative when the cancellation failed
	Message string `json:"msg"`
}

// WebsocketCancelReplaceNewOrderResponse holds the new order half of a cancel and replace order: the placed order, or
// the error code and message of a failed placement
type WebsocketCancelReplaceNewOrderResponse struct {
	WebsocketOrderResponse
	Code    int64  `json:"code"` // negative when the placement failed
	Message string `json:"msg"`
}

// WebsocketCancelOrderListRequest holds the parameters of the Cancel Order list (orderList.cancel) request; OrderListID
// or ListClientOrderID identifies the order list
type WebsocketCancelOrderListRequest struct {
	Symbol            currency.Pair
	OrderListID       uint64
	ListClientOrderID string
	// NewClientOrderID replaces the cancelled order list's client order list ID
	NewClientOrderID string
}

// WebsocketPlaceOneCancelsTheOtherOrderRequest holds the parameters of the Place new OCO - Deprecated (orderList.place)
// request
type WebsocketPlaceOneCancelsTheOtherOrderRequest struct {
	Symbol currency.Pair
	Side   Side
	// Price is the limit order's price
	Price                float64
	Quantity             float64
	ListClientOrderID    string
	LimitClientOrderID   string
	LimitIcebergQuantity float64
	LimitStrategyID      uint64
	LimitStrategyType    uint64
	// StopPrice or TrailingDeltaInBasisPoints, or both, trigger the stop order
	StopPrice                  float64
	TrailingDeltaInBasisPoints uint64
	StopClientOrderID          string
	StopLimitPrice             float64
	StopLimitTimeInForce       TimeInForce
	StopIcebergQuantity        float64
	StopStrategyID             uint64
	StopStrategyType           uint64
	NewOrderResponseType       NewOrderResponseType
	SelfTradePreventionMode    SelfTradePreventionMode
}

// WebsocketOrderListResponse is the Place new OCO - Deprecated (orderList.place) result: the order list with each leg's
// report
type WebsocketOrderListResponse struct {
	OrderListID       uint64                    `json:"orderListId"`
	ContingencyType   string                    `json:"contingencyType"`
	ListStatusType    string                    `json:"listStatusType"`
	ListOrderStatus   string                    `json:"listOrderStatus"`
	ListClientOrderID string                    `json:"listClientOrderId"`
	TransactionTime   types.Time                `json:"transactionTime"`
	Symbol            string                    `json:"symbol"`
	Orders            []WebsocketOrderListOrder `json:"orders"`
	OrderReports      []WebsocketOrderResponse  `json:"orderReports"`
}

// WebsocketPlaceOrderRequest holds the parameters of the Place new order (order.place) and Test new order (order.test)
// requests; the order type decides which of the optional parameters are mandatory
type WebsocketPlaceOrderRequest struct {
	Symbol      currency.Pair
	Side        Side
	Type        OrderType
	TimeInForce TimeInForce
	Price       float64
	Quantity    float64
	// QuoteOrderQuantity is the quote amount a MARKET order spends or receives instead of a base Quantity
	QuoteOrderQuantity float64
	NewClientOrderID   string
	// NewOrderResponseType is ACK, RESULT or FULL; MARKET and LIMIT orders default to FULL and the others to ACK
	NewOrderResponseType       NewOrderResponseType
	StopPrice                  float64
	TrailingDeltaInBasisPoints uint64
	IcebergQuantity            float64
	StrategyID                 uint64
	// StrategyType values below 1000000 are reserved
	StrategyType            uint64
	SelfTradePreventionMode SelfTradePreventionMode
	// PegPriceType is PRIMARY_PEG or MARKET_PEG
	PegPriceType PegPriceType
	// PegOffsetValue is the price level to peg to, at most 100
	PegOffsetValue uint64
	// PegOffsetType is PRICE_LEVEL
	PegOffsetType PegOffsetType
}

// WebsocketOrderFill is a trade that filled an order on placement
type WebsocketOrderFill struct {
	Price           types.Number  `json:"price"`
	Quantity        types.Number  `json:"qty"`
	Commission      types.Number  `json:"commission"`
	CommissionAsset currency.Code `json:"commissionAsset"`
	TradeID         uint64        `json:"tradeId"`
}

// WebsocketOrderResponse is the Place new order (order.place) result. ACK responses carry the identifying fields only,
// RESULT adds the order's state and FULL also the fills; the conditional fields appear only for orders they apply to
type WebsocketOrderResponse struct {
	Symbol                     string       `json:"symbol"`
	OrderID                    uint64       `json:"orderId"`
	OrderListID                int64        `json:"orderListId"` // -1 unless the order is an order list leg
	ClientOrderID              string       `json:"clientOrderId"`
	TransactTime               types.Time   `json:"transactTime"`
	Price                      types.Number `json:"price"`
	OriginalQuantity           types.Number `json:"origQty"`
	ExecutedQuantity           types.Number `json:"executedQty"`
	OriginalQuoteOrderQuantity types.Number `json:"origQuoteOrderQty"`
	CumulativeQuoteQuantity    types.Number `json:"cummulativeQuoteQty"`
	Status                     string       `json:"status"`
	TimeInForce                TimeInForce  `json:"timeInForce"`
	Type                       OrderType    `json:"type"`
	Side                       Side         `json:"side"`
	// WorkingTime is when the order started working on the book, in milliseconds; -1 for a pending order list leg
	WorkingTime                int64                   `json:"workingTime"`
	SelfTradePreventionMode    SelfTradePreventionMode `json:"selfTradePreventionMode"`
	Fills                      []WebsocketOrderFill    `json:"fills"`
	IcebergQuantity            types.Number            `json:"icebergQty"`
	PreventedMatchID           uint64                  `json:"preventedMatchId"`
	PreventedQuantity          types.Number            `json:"preventedQuantity"`
	StopPrice                  types.Number            `json:"stopPrice"`
	StrategyID                 uint64                  `json:"strategyId"`
	StrategyType               uint64                  `json:"strategyType"`
	TrailingDeltaInBasisPoints uint64                  `json:"trailingDelta"`
	// TrailingTime is when a trailing stop order started tracking the price, in milliseconds; -1 before it activates
	TrailingTime   int64         `json:"trailingTime"`
	UsedSOR        bool          `json:"usedSor"`
	WorkingFloor   string        `json:"workingFloor"`
	PegPriceType   PegPriceType  `json:"pegPriceType"`
	PegOffsetType  PegOffsetType `json:"pegOffsetType"`
	PegOffsetValue uint64        `json:"pegOffsetValue"`
	PeggedPrice    types.Number  `json:"peggedPrice"`
	ExpiryReason   string        `json:"expiryReason"`
}

// WebsocketOrderCommissionRates are an order's commission rates by trade role
type WebsocketOrderCommissionRates struct {
	Maker types.Number `json:"maker"`
	Taker types.Number `json:"taker"`
}

// WebsocketCommissionDiscount is the discount on standard commissions paid in the discount asset (BNB)
type WebsocketCommissionDiscount struct {
	EnabledForAccount bool          `json:"enabledForAccount"`
	EnabledForSymbol  bool          `json:"enabledForSymbol"`
	DiscountAsset     currency.Code `json:"discountAsset"`
	// Discount is the rate standard commissions are reduced by when paid in the discount asset
	Discount types.Number `json:"discount"`
}

// WebsocketOrderCommissionRatesResponse is the Test new order (order.test) result when commission rates are computed
type WebsocketOrderCommissionRatesResponse struct {
	StandardCommissionForOrder WebsocketOrderCommissionRates `json:"standardCommissionForOrder"`
	SpecialCommissionForOrder  WebsocketOrderCommissionRates `json:"specialCommissionForOrder"`
	TaxCommissionForOrder      WebsocketOrderCommissionRates `json:"taxCommissionForOrder"`
	Discount                   WebsocketCommissionDiscount   `json:"discount"`
}

// WebsocketSOROrderCommissionRatesResponse is the Test new order using SOR (sor.order.test) result when commission
// rates are computed
type WebsocketSOROrderCommissionRatesResponse struct {
	StandardCommissionForOrder WebsocketOrderCommissionRates `json:"standardCommissionForOrder"`
	TaxCommissionForOrder      WebsocketOrderCommissionRates `json:"taxCommissionForOrder"`
	Discount                   WebsocketCommissionDiscount   `json:"discount"`
}

// WebsocketSOROrderRequest holds the parameters of the Place new order using SOR (sor.order.place) and Test new order
// using SOR (sor.order.test) requests; only LIMIT and MARKET orders are supported
type WebsocketSOROrderRequest struct {
	Symbol currency.Pair
	Side   Side
	Type   OrderType
	// TimeInForce and Price apply to LIMIT orders only
	TimeInForce      TimeInForce
	Price            float64
	Quantity         float64
	NewClientOrderID string
	// NewOrderResponseType is ACK, RESULT or FULL; MARKET and LIMIT orders default to FULL
	NewOrderResponseType    NewOrderResponseType
	IcebergQuantity         float64
	StrategyID              uint64
	StrategyType            uint64
	SelfTradePreventionMode SelfTradePreventionMode
}

// WebsocketSOROrderFill is a trade that filled an SOR order on placement
type WebsocketSOROrderFill struct {
	MatchType       string        `json:"matchType"`
	Price           types.Number  `json:"price"`
	Quantity        types.Number  `json:"qty"`
	Commission      types.Number  `json:"commission"`
	CommissionAsset currency.Code `json:"commissionAsset"`
	TradeID         int64         `json:"tradeId"` // -1 for a fill from an allocation
	AllocationID    uint64        `json:"allocId"`
}

// WebsocketSOROrderResponse is a Place new order using SOR (sor.order.place) result entry
type WebsocketSOROrderResponse struct {
	Symbol                     string       `json:"symbol"`
	OrderID                    uint64       `json:"orderId"`
	OrderListID                int64        `json:"orderListId"` // always -1
	ClientOrderID              string       `json:"clientOrderId"`
	TransactTime               types.Time   `json:"transactTime"`
	Price                      types.Number `json:"price"`
	OriginalQuantity           types.Number `json:"origQty"`
	ExecutedQuantity           types.Number `json:"executedQty"`
	OriginalQuoteOrderQuantity types.Number `json:"origQuoteOrderQty"`
	CumulativeQuoteQuantity    types.Number `json:"cummulativeQuoteQty"`
	Status                     string       `json:"status"`
	TimeInForce                TimeInForce  `json:"timeInForce"`
	Type                       OrderType    `json:"type"`
	Side                       Side         `json:"side"`
	// WorkingTime is when the order started working, in milliseconds
	WorkingTime                int64                   `json:"workingTime"`
	Fills                      []WebsocketSOROrderFill `json:"fills"`
	WorkingFloor               string                  `json:"workingFloor"`
	SelfTradePreventionMode    SelfTradePreventionMode `json:"selfTradePreventionMode"`
	UsedSOR                    bool                    `json:"usedSor"`
	StopPrice                  types.Number            `json:"stopPrice"`
	TrailingDeltaInBasisPoints uint64                  `json:"trailingDelta"`
	IcebergQuantity            types.Number            `json:"icebergQty"`
	StrategyID                 uint64                  `json:"strategyId"`
	StrategyType               uint64                  `json:"strategyType"`
	PreventedMatchID           uint64                  `json:"preventedMatchId"`
	PreventedQuantity          types.Number            `json:"preventedQuantity"`
	// TrailingTime is when a trailing stop order started tracking the price, in milliseconds; -1 before it activates
	TrailingTime   int64         `json:"trailingTime"`
	PegPriceType   PegPriceType  `json:"pegPriceType"`
	PegOffsetType  PegOffsetType `json:"pegOffsetType"`
	PegOffsetValue uint64        `json:"pegOffsetValue"`
	PeggedPrice    types.Number  `json:"peggedPrice"`
	ExpiryReason   string        `json:"expiryReason"`
}

// WebsocketAccountCommissionRates are an account's commission rates by trade role
type WebsocketAccountCommissionRates struct {
	Maker  types.Number `json:"maker"`
	Taker  types.Number `json:"taker"`
	Buyer  types.Number `json:"buyer"`
	Seller types.Number `json:"seller"`
}

// WebsocketAccountCommissionResponse is the Account Commission Rates (account.commission) result
type WebsocketAccountCommissionResponse struct {
	Symbol             string                          `json:"symbol"`
	StandardCommission WebsocketAccountCommissionRates `json:"standardCommission"`
	SpecialCommission  WebsocketAccountCommissionRates `json:"specialCommission"`
	TaxCommission      WebsocketAccountCommissionRates `json:"taxCommission"`
	Discount           WebsocketCommissionDiscount     `json:"discount"`
}

// WebsocketOrderListHistoryRequest holds the parameters of the Account order list history (allOrderLists) request;
// FromID is ignored when StartTime or EndTime is set, and the window cannot exceed 24 hours
type WebsocketOrderListHistoryRequest struct {
	FromID    uint64
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
}

// WebsocketOrderHistoryRequest holds the parameters of the Account order history (allOrders) request; OrderID is
// ignored when StartTime or EndTime is set, and the window cannot exceed 24 hours
type WebsocketOrderHistoryRequest struct {
	Symbol currency.Pair
	// OrderID is the order ID to begin at
	OrderID   uint64
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
}

// WebsocketAllocationsRequest holds the parameters of the Account allocations (myAllocations) request; the window
// cannot exceed 24 hours
type WebsocketAllocationsRequest struct {
	Symbol           currency.Pair
	StartTime        time.Time
	EndTime          time.Time
	FromAllocationID uint64
	Limit            uint64
	OrderID          uint64
}

// WebsocketAllocationResponse is an Account allocations (myAllocations) result entry: an allocation from SOR order
// placement
type WebsocketAllocationResponse struct {
	Symbol          string        `json:"symbol"`
	AllocationID    uint64        `json:"allocationId"`
	AllocationType  string        `json:"allocationType"`
	OrderID         uint64        `json:"orderId"`
	OrderListID     int64         `json:"orderListId"` // -1 unless the order is an order list leg
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

// WebsocketPreventedMatchesRequest holds the parameters of the Account prevented matches (myPreventedMatches) request;
// it queries by PreventedMatchID, or by OrderID optionally paged with FromPreventedMatchID and Limit
type WebsocketPreventedMatchesRequest struct {
	Symbol               currency.Pair
	PreventedMatchID     uint64
	OrderID              uint64
	FromPreventedMatchID uint64
	Limit                uint64
}

// WebsocketPreventedMatchResponse is an Account prevented matches (myPreventedMatches) result entry: an order expired
// by self-trade prevention
type WebsocketPreventedMatchResponse struct {
	Symbol                  string                  `json:"symbol"`
	PreventedMatchID        uint64                  `json:"preventedMatchId"`
	TakerOrderID            uint64                  `json:"takerOrderId"`
	MakerSymbol             string                  `json:"makerSymbol"`
	MakerOrderID            uint64                  `json:"makerOrderId"`
	TradeGroupID            uint64                  `json:"tradeGroupId"`
	SelfTradePreventionMode SelfTradePreventionMode `json:"selfTradePreventionMode"`
	Price                   types.Number            `json:"price"`
	MakerPreventedQuantity  types.Number            `json:"makerPreventedQuantity"`
	TransactTime            types.Time              `json:"transactTime"`
}

// WebsocketAccountTradesRequest holds the parameters of the Account trade history (myTrades) request; OrderID cannot be
// combined with StartTime and EndTime, FromID cannot be combined with them either, and the window cannot exceed
// 24 hours
type WebsocketAccountTradesRequest struct {
	Symbol    currency.Pair
	OrderID   uint64
	StartTime time.Time
	EndTime   time.Time
	// FromID is the first trade ID to return
	FromID uint64
	Limit  uint64
}

// WebsocketAccountTradeResponse is an Account trade history (myTrades) result entry
type WebsocketAccountTradeResponse struct {
	Symbol          string        `json:"symbol"`
	ID              uint64        `json:"id"`
	OrderID         uint64        `json:"orderId"`
	OrderListID     int64         `json:"orderListId"` // -1 unless the order is an order list leg
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

// WebsocketUserDataStreamSubscriptionResponse is the result of a User Data Stream subscription: the ID its events carry
type WebsocketUserDataStreamSubscriptionResponse struct {
	SubscriptionID uint64 `json:"subscriptionId"`
}

// WebsocketListenTokenSubscriptionResponse is the userDataStream.subscribe.listenToken result
type WebsocketListenTokenSubscriptionResponse struct {
	SubscriptionID uint64 `json:"subscriptionId"`
	// ExpirationTime is when the subscription ends unless it is extended with a new listen token
	ExpirationTime types.Time `json:"expirationTime"`
}

// WebsocketAccountPosition is the outboundAccountPosition event, sent whenever an account balance changes with the
// balances the change may have affected
type WebsocketAccountPosition struct {
	EventType             string                    `json:"e"`
	EventTime             types.Time                `json:"E"`
	LastAccountUpdateTime types.Time                `json:"u"`
	Balances              []WebsocketAccountBalance `json:"B"`
}

// WebsocketAccountBalance is an asset balance of an outboundAccountPosition event
type WebsocketAccountBalance struct {
	Asset  currency.Code `json:"a"`
	Free   types.Number  `json:"f"`
	Locked types.Number  `json:"l"`
}

// WebsocketBalanceUpdate is the balanceUpdate event, sent on deposits, withdrawals and transfers between accounts
type WebsocketBalanceUpdate struct {
	EventType    string        `json:"e"`
	EventTime    types.Time    `json:"E"`
	Asset        currency.Code `json:"a"`
	BalanceDelta types.Number  `json:"d"`
	ClearTime    types.Time    `json:"T"`
}

// WebsocketExecutionReport is the executionReport event, sent whenever an order changes; the conditional fields appear
// only for orders they apply to
type WebsocketExecutionReport struct {
	EventType       string       `json:"e"`
	EventTime       types.Time   `json:"E"`
	Symbol          string       `json:"s"`
	ClientOrderID   string       `json:"c"`
	Side            Side         `json:"S"`
	OrderType       OrderType    `json:"o"`
	TimeInForce     TimeInForce  `json:"f"`
	OrderQuantity   types.Number `json:"q"`
	OrderPrice      types.Number `json:"p"`
	StopPrice       types.Number `json:"P"`
	IcebergQuantity types.Number `json:"F"`
	OrderListID     int64        `json:"g"` // -1 unless the order is an order list leg
	// OriginalClientOrderID is the client order ID of the order being cancelled
	OriginalClientOrderID    string        `json:"C"`
	CurrentExecutionType     string        `json:"x"`
	CurrentOrderStatus       string        `json:"X"`
	OrderRejectReason        string        `json:"r"`
	OrderID                  uint64        `json:"i"`
	LastExecutedQuantity     types.Number  `json:"l"`
	CumulativeFilledQuantity types.Number  `json:"z"`
	LastExecutedPrice        types.Number  `json:"L"`
	CommissionAmount         types.Number  `json:"n"`
	CommissionAsset          currency.Code `json:"N"`
	TransactionTime          types.Time    `json:"T"`
	TradeID                  int64         `json:"t"` // -1 unless the event is a trade
	PreventedMatchID         uint64        `json:"v"`
	ExecutionID              uint64        `json:"I"`
	IsOrderOnBook            bool          `json:"w"`
	IsMaker                  bool          `json:"m"`
	Ignore                   bool          `json:"M"`
	OrderCreationTime        types.Time    `json:"O"`
	// CumulativeQuoteAssetTransactedQuantity divided by CumulativeFilledQuantity is the average price
	CumulativeQuoteAssetTransactedQuantity types.Number            `json:"Z"`
	LastQuoteAssetTransactedQuantity       types.Number            `json:"Y"`
	QuoteOrderQuantity                     types.Number            `json:"Q"`
	WorkingTime                            types.Time              `json:"W"`
	SelfTradePreventionMode                SelfTradePreventionMode `json:"V"`
	TrailingDeltaInBasisPoints             uint64                  `json:"d"`
	// TrailingTime is when a trailing stop order started tracking the price, in milliseconds; -1 before it activates
	TrailingTime                    int64         `json:"D"`
	StrategyID                      uint64        `json:"j"`
	StrategyType                    uint64        `json:"J"`
	PreventedQuantity               types.Number  `json:"A"`
	LastPreventedQuantity           types.Number  `json:"B"`
	TradeGroupID                    uint64        `json:"u"`
	CounterOrderID                  uint64        `json:"U"`
	CounterSymbol                   string        `json:"Cs"`
	PreventedExecutionQuantity      types.Number  `json:"pl"`
	PreventedExecutionPrice         types.Number  `json:"pL"`
	PreventedExecutionQuoteQuantity types.Number  `json:"pY"`
	MatchType                       string        `json:"b"`
	AllocationID                    uint64        `json:"a"`
	WorkingFloor                    string        `json:"k"`
	UsedSOR                         bool          `json:"uS"`
	PeggedPriceType                 PegPriceType  `json:"gP"`
	PeggedOffsetType                PegOffsetType `json:"gOT"`
	PeggedOffsetValue               uint64        `json:"gOV"`
	PeggedPrice                     types.Number  `json:"gp"`
	ExpiryReason                    string        `json:"eR"`
}

// WebsocketListStatus is the listStatus event, sent with the executionReport events of an order list's legs
type WebsocketListStatus struct {
	EventType         string                     `json:"e"`
	EventTime         types.Time                 `json:"E"`
	Symbol            string                     `json:"s"`
	OrderListID       uint64                     `json:"g"`
	ContingencyType   string                     `json:"c"`
	ListStatusType    string                     `json:"l"`
	ListOrderStatus   string                     `json:"L"`
	ListRejectReason  string                     `json:"r"`
	ListClientOrderID string                     `json:"C"`
	TransactionTime   types.Time                 `json:"T"`
	Orders            []WebsocketListStatusOrder `json:"O"`
}

// WebsocketListStatusOrder is an order list leg of a listStatus event
type WebsocketListStatusOrder struct {
	Symbol        string `json:"s"`
	OrderID       uint64 `json:"i"`
	ClientOrderID string `json:"c"`
}

// WebsocketExternalLockUpdate is the externalLockUpdate event, sent when an external system locks or unlocks part of a
// spot balance, for example as margin collateral
type WebsocketExternalLockUpdate struct {
	EventType       string        `json:"e"`
	EventTime       types.Time    `json:"E"`
	Asset           currency.Code `json:"a"`
	Delta           types.Number  `json:"d"`
	TransactionTime types.Time    `json:"T"`
}

// WebsocketEventStreamTerminated is the eventStreamTerminated event, sent when a User Data Stream subscription ends: a
// listen token expired or the subscription was stopped
type WebsocketEventStreamTerminated struct {
	EventType string     `json:"e"`
	EventTime types.Time `json:"E"`
}
