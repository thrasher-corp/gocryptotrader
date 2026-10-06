package binance

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// WsPayload is a WebSocket Streams subscription request: SUBSCRIBE or UNSUBSCRIBE with the stream names
type WsPayload struct {
	Method string   `json:"method"`
	Params []string `json:"params"`
	ID     string   `json:"id"`
}

// WsServerShutdown is the serverShutdown event, sent on the stream and WebSocket API connections when the server is
// about to shut down
type WsServerShutdown struct {
	EventType string     `json:"e"`
	EventTime types.Time `json:"E"`
}

// WsPartialDepth is a Partial Book Depth Streams (<symbol>@depth<levels>) payload: the top levels of the book
type WsPartialDepth struct {
	// LastUpdateID is signed like the order book's update IDs it is loaded into
	LastUpdateID int64                            `json:"lastUpdateId"`
	Bids         orderbook.LevelsArrayPriceAmount `json:"bids"`
	Asks         orderbook.LevelsArrayPriceAmount `json:"asks"`
}

// DiffDepthStream is a Diff. Depth Stream (<symbol>@depth) payload: the new quantity of each changed price level.
// Update IDs are signed like the order book's update IDs they feed
type DiffDepthStream struct {
	EventType     string                           `json:"e"`
	EventTime     types.Time                       `json:"E"`
	Symbol        string                           `json:"s"`
	FirstUpdateID int64                            `json:"U"`
	FinalUpdateID int64                            `json:"u"`
	Bids          orderbook.LevelsArrayPriceAmount `json:"b"`
	Asks          orderbook.LevelsArrayPriceAmount `json:"a"`
}

// KlineStream is a Kline/Candlestick Streams (<symbol>@kline_<interval>) payload
type KlineStream struct {
	EventType string          `json:"e"`
	EventTime types.Time      `json:"E"`
	Symbol    string          `json:"s"`
	Kline     KlineStreamData `json:"k"`
}

// KlineStreamData is the candle of a kline stream payload
type KlineStreamData struct {
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

// WsAPIRequest is the envelope of a WebSocket API request
type WsAPIRequest struct {
	ID     string         `json:"id"`
	Method string         `json:"method"`
	Params map[string]any `json:"params,omitempty"`
}

// WsAPIResponse is the envelope of a WebSocket API response; Result is present when the request succeeded and Error
// when it failed
type WsAPIResponse struct {
	ID         string          `json:"id"`
	Status     int64           `json:"status"`
	Result     json.RawMessage `json:"result"`
	Error      *WsAPIError     `json:"error"`
	RateLimits []WsRateLimit   `json:"rateLimits"`
}

// WsAPIError is a failed WebSocket API request's error
type WsAPIError struct {
	Code    int64  `json:"code"`
	Message string `json:"msg"`
	// Data holds method specific details, such as retryAfter when rate limited or the outcome of each half of a
	// cancel-replace
	Data json.RawMessage `json:"data"`
}

// WsRateLimit is a rate limit's current usage
type WsRateLimit struct {
	RateLimitType  string `json:"rateLimitType"`
	Interval       string `json:"interval"`
	IntervalNumber uint64 `json:"intervalNum"`
	Limit          uint64 `json:"limit"`
	Count          uint64 `json:"count"`
}

// WsAveragePriceResponse is the Current average price (avgPrice) result
type WsAveragePriceResponse struct {
	Minutes uint64       `json:"mins"` // Average price interval in minutes
	Price   types.Number `json:"price"`
	// CloseTime is the time of the last trade
	CloseTime types.Time `json:"closeTime"`
}

// WsKlinesRequest holds the parameters of the Klines (klines) and UI Klines (uiKlines) requests
type WsKlinesRequest struct {
	Symbol    currency.Pair
	Interval  kline.Interval
	StartTime time.Time
	EndTime   time.Time
	// TimeZone is the offset klines are interpreted in, e.g. "-1:00", "05:45" or "8"; UTC when empty
	TimeZone string
	Limit    uint64
}

// WsRollingWindowTickerRequest holds the parameters of the Rolling window price change statistics (ticker) request
type WsRollingWindowTickerRequest struct {
	// Symbols are sent as symbol when there is one and as symbols when there are several
	Symbols currency.Pairs
	// WindowSize is a whole number of minutes (1 to 59), hours (1 to 23) or days (1 to 7); 1 day when zero
	WindowSize time.Duration
	// Type is FULL (the default) or MINI
	Type string
	// SymbolStatus filters the symbols by trading status: TRADING, HALT or BREAK
	SymbolStatus string
}

// WsTickerRequest holds the parameters of the 24hr ticker price change statistics (ticker.24hr) request
type WsTickerRequest struct {
	// Symbols are sent as symbol when there is one and as symbols when there are several; every symbol when empty
	Symbols currency.Pairs
	// Type is FULL (the default) or MINI
	Type string
	// SymbolStatus filters the symbols by trading status: TRADING, HALT or BREAK
	SymbolStatus string
}

// WsBookTicker is a Symbol order book ticker (ticker.book) result entry
type WsBookTicker struct {
	Symbol      string       `json:"symbol"`
	BidPrice    types.Number `json:"bidPrice"`
	BidQuantity types.Number `json:"bidQty"`
	AskPrice    types.Number `json:"askPrice"`
	AskQuantity types.Number `json:"askQty"`
}

// WsPriceTicker is a Symbol price ticker (ticker.price) result entry
type WsPriceTicker struct {
	Symbol string       `json:"symbol"`
	Price  types.Number `json:"price"`
}

// WsTradingDayTickerRequest holds the parameters of the Trading Day Ticker (ticker.tradingDay) request
type WsTradingDayTickerRequest struct {
	// Symbols are sent as symbol when there is one and as symbols when there are several
	Symbols currency.Pairs
	// TimeZone is the offset the trading day starts in, e.g. "-1:00", "05:45" or "8"; UTC when empty
	TimeZone string
	// Type is FULL (the default) or MINI
	Type string
	// SymbolStatus filters the symbols by trading status: TRADING, HALT or BREAK
	SymbolStatus string
}

// WsAggregatedTradesRequest holds the parameters of the Aggregate trades (trades.aggregate) request; FromID cannot be
// combined with StartTime and EndTime
type WsAggregatedTradesRequest struct {
	Symbol    currency.Pair
	FromID    uint64
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
}

// WsSessionStatusResponse is the Query session status (session.status) and Log out of the session (session.logout)
// result
type WsSessionStatusResponse struct {
	// APIKey is empty when the session is not authenticated
	APIKey string `json:"apiKey"`
	// AuthorizedSince is zero when the session is not authenticated
	AuthorizedSince  types.Time `json:"authorizedSince"`
	ConnectedSince   types.Time `json:"connectedSince"`
	ReturnRateLimits bool       `json:"returnRateLimits"`
	ServerTime       types.Time `json:"serverTime"`
	// UserDataStream reports whether the session has an active User Data Stream subscription; nil when Binance sends
	// null or leaves it out, so an unknown state is not mistaken for no subscription
	UserDataStream *bool `json:"userDataStream"`
}

// WsOrderReport is an order's state as order cancellation results report it; the conditional fields appear only for
// orders they apply to
type WsOrderReport struct {
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
	CummulativeQuoteQuantity   types.Number `json:"cummulativeQuoteQty"`
	Status                     string       `json:"status"`
	TimeInForce                string       `json:"timeInForce"`
	Type                       string       `json:"type"`
	Side                       string       `json:"side"`
	StopPrice                  types.Number `json:"stopPrice"`
	TrailingDelta              uint64       `json:"trailingDelta"`
	// TrailingTime is when a trailing stop order started tracking the price, in milliseconds; -1 before it activates
	TrailingTime            int64        `json:"trailingTime"`
	IcebergQuantity         types.Number `json:"icebergQty"`
	StrategyID              int64        `json:"strategyId"`
	StrategyType            int64        `json:"strategyType"`
	SelfTradePreventionMode string       `json:"selfTradePreventionMode"`
	PreventedMatchID        uint64       `json:"preventedMatchId"`
	PreventedQuantity       types.Number `json:"preventedQuantity"`
	UsedSOR                 bool         `json:"usedSor"`
	WorkingFloor            string       `json:"workingFloor"`
	PegPriceType            string       `json:"pegPriceType"`
	PegOffsetType           string       `json:"pegOffsetType"`
	PegOffsetValue          uint64       `json:"pegOffsetValue"`
	PeggedPrice             types.Number `json:"peggedPrice"`
	ExpiryReason            string       `json:"expiryReason"`
}

// WsOrderListOrder is an order list's leg
type WsOrderListOrder struct {
	Symbol        string `json:"symbol"`
	OrderID       uint64 `json:"orderId"`
	ClientOrderID string `json:"clientOrderId"`
}

// WsCancelOrderResponse is the Cancel order (order.cancel) and Cancel Order list (orderList.cancel) result and a
// Cancel open orders (openOrders.cancelAll) result entry. Cancelling an order list, or one of its legs, cancels the
// whole list, which is reported with the list fields and each leg's report instead of the order fields
type WsCancelOrderResponse struct {
	WsOrderReport
	ContingencyType   string             `json:"contingencyType"`
	ListStatusType    string             `json:"listStatusType"`
	ListOrderStatus   string             `json:"listOrderStatus"`
	ListClientOrderID string             `json:"listClientOrderId"`
	TransactionTime   types.Time         `json:"transactionTime"`
	Orders            []WsOrderListOrder `json:"orders"`
	OrderReports      []WsOrderReport    `json:"orderReports"`
}

// WsCancelOrderRequest holds the parameters of the Cancel order (order.cancel) request; OrderID or
// OriginalClientOrderID identifies the order
type WsCancelOrderRequest struct {
	Symbol                currency.Pair
	OrderID               uint64
	OriginalClientOrderID string
	// NewClientOrderID replaces the cancelled order's client order ID, freeing it for new orders
	NewClientOrderID string
	// CancelRestrictions limits the cancellation to orders with status NEW (ONLY_NEW) or PARTIALLY_FILLED
	// (ONLY_PARTIALLY_FILLED)
	CancelRestrictions string
}

// WsCancelReplaceOrderRequest holds the parameters of the Cancel and replace order (order.cancelReplace) request;
// CancelOrderID or CancelOriginalClientOrderID identifies the order to cancel
type WsCancelReplaceOrderRequest struct {
	Symbol currency.Pair
	// CancelReplaceMode is STOP_ON_FAILURE or ALLOW_FAILURE
	CancelReplaceMode           string
	CancelOrderID               uint64
	CancelOriginalClientOrderID string
	CancelNewClientOrderID      string
	Side                        string
	Type                        string
	TimeInForce                 string
	Price                       float64
	Quantity                    float64
	QuoteOrderQuantity          float64
	NewClientOrderID            string
	// NewOrderResponseType is ACK, RESULT or FULL
	NewOrderResponseType    string
	StopPrice               float64
	TrailingDelta           uint64
	IcebergQuantity         float64
	StrategyID              int64
	StrategyType            int64
	SelfTradePreventionMode string
	CancelRestrictions      string
	// OrderRateLimitExceededMode is DO_NOTHING (the default) or CANCEL_ONLY
	OrderRateLimitExceededMode string
	PegPriceType               string
	PegOffsetValue             uint64
	PegOffsetType              string
}

// WsCancelReplaceOrderResponse is the Cancel and replace order (order.cancelReplace) result. Binance replies with it
// when both halves succeed and puts it in the error's data when either fails, so a failed half holds an error code and
// message instead of an order
type WsCancelReplaceOrderResponse struct {
	// CancelResult is SUCCESS or FAILURE
	CancelResult string `json:"cancelResult"`
	// NewOrderResult is SUCCESS, FAILURE or NOT_ATTEMPTED
	NewOrderResult   string                          `json:"newOrderResult"`
	CancelResponse   WsCancelReplaceCancelResponse   `json:"cancelResponse"`
	NewOrderResponse WsCancelReplaceNewOrderResponse `json:"newOrderResponse"`
}

// WsCancelReplaceCancelResponse holds the cancellation half of a cancel and replace order: the cancelled order, or the
// error code and message of a failed cancellation
type WsCancelReplaceCancelResponse struct {
	WsCancelOrderResponse
	Code    int64  `json:"code"`
	Message string `json:"msg"`
}

// WsCancelReplaceNewOrderResponse holds the new order half of a cancel and replace order: the placed order, or the
// error code and message of a failed placement
type WsCancelReplaceNewOrderResponse struct {
	WsOrderResponse
	Code    int64  `json:"code"`
	Message string `json:"msg"`
}

// WsCancelOrderListRequest holds the parameters of the Cancel Order list (orderList.cancel) request; OrderListID or
// ListClientOrderID identifies the order list
type WsCancelOrderListRequest struct {
	Symbol            currency.Pair
	OrderListID       uint64
	ListClientOrderID string
	// NewClientOrderID replaces the cancelled order list's client order list ID
	NewClientOrderID string
}

// WsPlaceOCOOrderRequest holds the parameters of the Place new OCO - Deprecated (orderList.place) request
type WsPlaceOCOOrderRequest struct {
	Symbol currency.Pair
	Side   string
	// Price is the limit order's price
	Price                float64
	Quantity             float64
	ListClientOrderID    string
	LimitClientOrderID   string
	LimitIcebergQuantity float64
	LimitStrategyID      int64
	LimitStrategyType    int64
	// StopPrice or TrailingDelta, or both, trigger the stop order
	StopPrice               float64
	TrailingDelta           uint64
	StopClientOrderID       string
	StopLimitPrice          float64
	StopLimitTimeInForce    string
	StopIcebergQuantity     float64
	StopStrategyID          int64
	StopStrategyType        int64
	NewOrderResponseType    string
	SelfTradePreventionMode string
}

// WsOrderListResponse is the Place new OCO - Deprecated (orderList.place) result: the order list with each leg's report
type WsOrderListResponse struct {
	OrderListID       uint64             `json:"orderListId"`
	ContingencyType   string             `json:"contingencyType"`
	ListStatusType    string             `json:"listStatusType"`
	ListOrderStatus   string             `json:"listOrderStatus"`
	ListClientOrderID string             `json:"listClientOrderId"`
	TransactionTime   types.Time         `json:"transactionTime"`
	Symbol            string             `json:"symbol"`
	Orders            []WsOrderListOrder `json:"orders"`
	OrderReports      []WsOrderResponse  `json:"orderReports"`
}

// WsPlaceOrderRequest holds the parameters of the Place new order (order.place) and Test new order (order.test)
// requests; the order type decides which of the optional parameters are mandatory
type WsPlaceOrderRequest struct {
	Symbol      currency.Pair
	Side        string
	Type        string
	TimeInForce string
	Price       float64
	Quantity    float64
	// QuoteOrderQuantity is the quote amount a MARKET order spends or receives instead of a base Quantity
	QuoteOrderQuantity float64
	NewClientOrderID   string
	// NewOrderResponseType is ACK, RESULT or FULL; MARKET and LIMIT orders default to FULL and the others to ACK
	NewOrderResponseType string
	StopPrice            float64
	TrailingDelta        uint64
	IcebergQuantity      float64
	StrategyID           int64
	// StrategyType values below 1000000 are reserved
	StrategyType            int64
	SelfTradePreventionMode string
	// PegPriceType is PRIMARY_PEG or MARKET_PEG
	PegPriceType string
	// PegOffsetValue is the price level to peg to, at most 100
	PegOffsetValue uint64
	// PegOffsetType is PRICE_LEVEL
	PegOffsetType string
}

// WsOrderFill is a trade that filled an order on placement
type WsOrderFill struct {
	Price           types.Number  `json:"price"`
	Quantity        types.Number  `json:"qty"`
	Commission      types.Number  `json:"commission"`
	CommissionAsset currency.Code `json:"commissionAsset"`
	TradeID         uint64        `json:"tradeId"`
}

// WsOrderResponse is the Place new order (order.place) result. ACK responses carry the identifying fields only, RESULT
// adds the order's state and FULL also the fills; the conditional fields appear only for orders they apply to
type WsOrderResponse struct {
	Symbol                     string       `json:"symbol"`
	OrderID                    uint64       `json:"orderId"`
	OrderListID                int64        `json:"orderListId"` // -1 unless the order is an order list leg
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
	// WorkingTime is when the order started working on the book, in milliseconds; -1 for a pending order list leg
	WorkingTime             int64         `json:"workingTime"`
	SelfTradePreventionMode string        `json:"selfTradePreventionMode"`
	Fills                   []WsOrderFill `json:"fills"`
	IcebergQuantity         types.Number  `json:"icebergQty"`
	PreventedMatchID        uint64        `json:"preventedMatchId"`
	PreventedQuantity       types.Number  `json:"preventedQuantity"`
	StopPrice               types.Number  `json:"stopPrice"`
	StrategyID              int64         `json:"strategyId"`
	StrategyType            int64         `json:"strategyType"`
	TrailingDelta           uint64        `json:"trailingDelta"`
	// TrailingTime is when a trailing stop order started tracking the price, in milliseconds; -1 before it activates
	TrailingTime   int64        `json:"trailingTime"`
	UsedSOR        bool         `json:"usedSor"`
	WorkingFloor   string       `json:"workingFloor"`
	PegPriceType   string       `json:"pegPriceType"`
	PegOffsetType  string       `json:"pegOffsetType"`
	PegOffsetValue uint64       `json:"pegOffsetValue"`
	PeggedPrice    types.Number `json:"peggedPrice"`
	ExpiryReason   string       `json:"expiryReason"`
}

// WsOrderCommissionRates are an order's commission rates by trade role
type WsOrderCommissionRates struct {
	Maker types.Number `json:"maker"`
	Taker types.Number `json:"taker"`
}

// WsCommissionDiscount is the discount on standard commissions paid in the discount asset (BNB)
type WsCommissionDiscount struct {
	EnabledForAccount bool          `json:"enabledForAccount"`
	EnabledForSymbol  bool          `json:"enabledForSymbol"`
	DiscountAsset     currency.Code `json:"discountAsset"`
	// Discount is the rate standard commissions are reduced by when paid in the discount asset
	Discount types.Number `json:"discount"`
}

// WsOrderCommissionRatesResponse is the Test new order (order.test) result when commission rates are computed
type WsOrderCommissionRatesResponse struct {
	StandardCommissionForOrder WsOrderCommissionRates `json:"standardCommissionForOrder"`
	SpecialCommissionForOrder  WsOrderCommissionRates `json:"specialCommissionForOrder"`
	TaxCommissionForOrder      WsOrderCommissionRates `json:"taxCommissionForOrder"`
	Discount                   WsCommissionDiscount   `json:"discount"`
}

// WsSOROrderCommissionRatesResponse is the Test new order using SOR (sor.order.test) result when commission rates are
// computed
type WsSOROrderCommissionRatesResponse struct {
	StandardCommissionForOrder WsOrderCommissionRates `json:"standardCommissionForOrder"`
	TaxCommissionForOrder      WsOrderCommissionRates `json:"taxCommissionForOrder"`
	Discount                   WsCommissionDiscount   `json:"discount"`
}

// WsSOROrderRequest holds the parameters of the Place new order using SOR (sor.order.place) and Test new order using
// SOR (sor.order.test) requests; only LIMIT and MARKET orders are supported
type WsSOROrderRequest struct {
	Symbol currency.Pair
	Side   string
	Type   string
	// TimeInForce and Price apply to LIMIT orders only
	TimeInForce      string
	Price            float64
	Quantity         float64
	NewClientOrderID string
	// NewOrderResponseType is ACK, RESULT or FULL; MARKET and LIMIT orders default to FULL
	NewOrderResponseType    string
	IcebergQuantity         float64
	StrategyID              int64
	StrategyType            int64
	SelfTradePreventionMode string
}

// WsSOROrderFill is a trade that filled an SOR order on placement
type WsSOROrderFill struct {
	MatchType       string        `json:"matchType"`
	Price           types.Number  `json:"price"`
	Quantity        types.Number  `json:"qty"`
	Commission      types.Number  `json:"commission"`
	CommissionAsset currency.Code `json:"commissionAsset"`
	TradeID         int64         `json:"tradeId"` // -1 for a fill from an allocation
	AllocationID    uint64        `json:"allocId"`
}

// WsSOROrderResponse is a Place new order using SOR (sor.order.place) result entry
type WsSOROrderResponse struct {
	Symbol                     string       `json:"symbol"`
	OrderID                    uint64       `json:"orderId"`
	OrderListID                int64        `json:"orderListId"` // always -1
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
	// WorkingTime is when the order started working, in milliseconds
	WorkingTime             int64            `json:"workingTime"`
	Fills                   []WsSOROrderFill `json:"fills"`
	WorkingFloor            string           `json:"workingFloor"`
	SelfTradePreventionMode string           `json:"selfTradePreventionMode"`
	UsedSOR                 bool             `json:"usedSor"`
	StopPrice               types.Number     `json:"stopPrice"`
	TrailingDelta           uint64           `json:"trailingDelta"`
	IcebergQuantity         types.Number     `json:"icebergQty"`
	StrategyID              int64            `json:"strategyId"`
	StrategyType            int64            `json:"strategyType"`
	PreventedMatchID        uint64           `json:"preventedMatchId"`
	PreventedQuantity       types.Number     `json:"preventedQuantity"`
	// TrailingTime is when a trailing stop order started tracking the price, in milliseconds; -1 before it activates
	TrailingTime   int64        `json:"trailingTime"`
	PegPriceType   string       `json:"pegPriceType"`
	PegOffsetType  string       `json:"pegOffsetType"`
	PegOffsetValue uint64       `json:"pegOffsetValue"`
	PeggedPrice    types.Number `json:"peggedPrice"`
	ExpiryReason   string       `json:"expiryReason"`
}

// WsAccountCommissionRates are an account's commission rates by trade role
type WsAccountCommissionRates struct {
	Maker  types.Number `json:"maker"`
	Taker  types.Number `json:"taker"`
	Buyer  types.Number `json:"buyer"`
	Seller types.Number `json:"seller"`
}

// WsAccountCommissionResponse is the Account Commission Rates (account.commission) result
type WsAccountCommissionResponse struct {
	Symbol             string                   `json:"symbol"`
	StandardCommission WsAccountCommissionRates `json:"standardCommission"`
	SpecialCommission  WsAccountCommissionRates `json:"specialCommission"`
	TaxCommission      WsAccountCommissionRates `json:"taxCommission"`
	Discount           WsCommissionDiscount     `json:"discount"`
}

// WsOrderListHistoryRequest holds the parameters of the Account order list history (allOrderLists) request; FromID is
// ignored when StartTime or EndTime is set, and the window cannot exceed 24 hours
type WsOrderListHistoryRequest struct {
	FromID    uint64
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
}

// WsOrderHistoryRequest holds the parameters of the Account order history (allOrders) request; OrderID is ignored
// when StartTime or EndTime is set, and the window cannot exceed 24 hours
type WsOrderHistoryRequest struct {
	Symbol currency.Pair
	// OrderID is the order ID to begin at
	OrderID   uint64
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
}

// WsAllocationsRequest holds the parameters of the Account allocations (myAllocations) request; the window cannot
// exceed 24 hours
type WsAllocationsRequest struct {
	Symbol           currency.Pair
	StartTime        time.Time
	EndTime          time.Time
	FromAllocationID uint64
	Limit            uint64
	OrderID          uint64
}

// WsAllocation is an Account allocations (myAllocations) result entry: an allocation from SOR order placement
type WsAllocation struct {
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

// WsPreventedMatchesRequest holds the parameters of the Account prevented matches (myPreventedMatches) request; it
// queries by PreventedMatchID, or by OrderID optionally paged with FromPreventedMatchID and Limit
type WsPreventedMatchesRequest struct {
	Symbol               currency.Pair
	PreventedMatchID     uint64
	OrderID              uint64
	FromPreventedMatchID uint64
	Limit                uint64
}

// WsPreventedMatch is an Account prevented matches (myPreventedMatches) result entry: an order expired by self-trade
// prevention
type WsPreventedMatch struct {
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

// WsAccountTradesRequest holds the parameters of the Account trade history (myTrades) request; OrderID cannot be
// combined with StartTime and EndTime, FromID cannot be combined with them either, and the window cannot exceed
// 24 hours
type WsAccountTradesRequest struct {
	Symbol    currency.Pair
	OrderID   uint64
	StartTime time.Time
	EndTime   time.Time
	// FromID is the first trade ID to return
	FromID uint64
	Limit  uint64
}

// WsAccountTrade is an Account trade history (myTrades) result entry
type WsAccountTrade struct {
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

// WsUserDataStreamSubscriptionResponse is the result of a User Data Stream subscription: the ID its events carry
type WsUserDataStreamSubscriptionResponse struct {
	SubscriptionID uint64 `json:"subscriptionId"`
}

// WsListenTokenSubscriptionResponse is the userDataStream.subscribe.listenToken result
type WsListenTokenSubscriptionResponse struct {
	SubscriptionID uint64 `json:"subscriptionId"`
	// ExpirationTime is when the subscription ends unless it is extended with a new listen token
	ExpirationTime types.Time `json:"expirationTime"`
}

// WsAccountPosition is the outboundAccountPosition event, sent whenever an account balance changes with the balances
// the change may have affected
type WsAccountPosition struct {
	EventType             string             `json:"e"`
	EventTime             types.Time         `json:"E"`
	LastAccountUpdateTime types.Time         `json:"u"`
	Balances              []WsAccountBalance `json:"B"`
}

// WsAccountBalance is an asset balance of an outboundAccountPosition event
type WsAccountBalance struct {
	Asset  currency.Code `json:"a"`
	Free   types.Number  `json:"f"`
	Locked types.Number  `json:"l"`
}

// WsBalanceUpdate is the balanceUpdate event, sent on deposits, withdrawals and transfers between accounts
type WsBalanceUpdate struct {
	EventType    string        `json:"e"`
	EventTime    types.Time    `json:"E"`
	Asset        currency.Code `json:"a"`
	BalanceDelta types.Number  `json:"d"`
	ClearTime    types.Time    `json:"T"`
}

// WsExecutionReport is the executionReport event, sent whenever an order changes; the conditional fields appear only
// for orders they apply to
type WsExecutionReport struct {
	EventType       string       `json:"e"`
	EventTime       types.Time   `json:"E"`
	Symbol          string       `json:"s"`
	ClientOrderID   string       `json:"c"`
	Side            string       `json:"S"`
	OrderType       string       `json:"o"`
	TimeInForce     string       `json:"f"`
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
	CumulativeQuoteAssetTransactedQuantity types.Number `json:"Z"`
	LastQuoteAssetTransactedQuantity       types.Number `json:"Y"`
	QuoteOrderQuantity                     types.Number `json:"Q"`
	WorkingTime                            types.Time   `json:"W"`
	SelfTradePreventionMode                string       `json:"V"`
	TrailingDelta                          uint64       `json:"d"`
	// TrailingTime is when a trailing stop order started tracking the price, in milliseconds; -1 before it activates
	TrailingTime                    int64        `json:"D"`
	StrategyID                      int64        `json:"j"`
	StrategyType                    int64        `json:"J"`
	PreventedQuantity               types.Number `json:"A"`
	LastPreventedQuantity           types.Number `json:"B"`
	TradeGroupID                    uint64       `json:"u"`
	CounterOrderID                  uint64       `json:"U"`
	CounterSymbol                   string       `json:"Cs"`
	PreventedExecutionQuantity      types.Number `json:"pl"`
	PreventedExecutionPrice         types.Number `json:"pL"`
	PreventedExecutionQuoteQuantity types.Number `json:"pY"`
	MatchType                       string       `json:"b"`
	AllocationID                    uint64       `json:"a"`
	WorkingFloor                    string       `json:"k"`
	UsedSOR                         bool         `json:"uS"`
	PeggedPriceType                 string       `json:"gP"`
	PeggedOffsetType                string       `json:"gOT"`
	PeggedOffsetValue               uint64       `json:"gOV"`
	PeggedPrice                     types.Number `json:"gp"`
	ExpiryReason                    string       `json:"eR"`
}

// WsListStatus is the listStatus event, sent with the executionReport events of an order list's legs
type WsListStatus struct {
	EventType         string              `json:"e"`
	EventTime         types.Time          `json:"E"`
	Symbol            string              `json:"s"`
	OrderListID       uint64              `json:"g"`
	ContingencyType   string              `json:"c"`
	ListStatusType    string              `json:"l"`
	ListOrderStatus   string              `json:"L"`
	ListRejectReason  string              `json:"r"`
	ListClientOrderID string              `json:"C"`
	TransactionTime   types.Time          `json:"T"`
	Orders            []WsListStatusOrder `json:"O"`
}

// WsListStatusOrder is an order list leg of a listStatus event
type WsListStatusOrder struct {
	Symbol        string `json:"s"`
	OrderID       uint64 `json:"i"`
	ClientOrderID string `json:"c"`
}

// WsExternalLockUpdate is the externalLockUpdate event, sent when an external system locks or unlocks part of a spot
// balance, for example as margin collateral
type WsExternalLockUpdate struct {
	EventType       string        `json:"e"`
	EventTime       types.Time    `json:"E"`
	Asset           currency.Code `json:"a"`
	Delta           types.Number  `json:"d"`
	TransactionTime types.Time    `json:"T"`
}

// WsEventStreamTerminated is the eventStreamTerminated event, sent when a User Data Stream subscription ends: a listen
// token expired or the subscription was stopped
type WsEventStreamTerminated struct {
	EventType string     `json:"e"`
	EventTime types.Time `json:"E"`
}
