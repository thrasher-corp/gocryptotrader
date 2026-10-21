package htx

import (
	"bytes"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Order price types referenced when mapping GoCryptoTrader order types onto the HTX API
const (
	orderPriceTypeLightning    = "lightning"
	orderPriceTypeLightningIOC = "lightning_ioc"
	orderTimeInForceFOK        = "fok"
	orderTimeInForceIOC        = "ioc"
	marginModeCross            = "cross"
	marginModeIsolated         = "isolated"
	orderPriceTypeLimit        = "limit"
	orderPriceTypeOpponent     = "opponent"
	orderPriceTypeOptimal5     = "optimal_5"
	orderPriceTypeOptimal10    = "optimal_10"
	orderPriceTypeOptimal20    = "optimal_20"
	orderPriceTypeOptimal20IOC = "optimal_20_ioc"
	orderPriceTypeOptimal20FOK = "optimal_20_fok"
	orderPriceTypePostOnly     = "post_only"
)

type errorCapture struct {
	Status      string     `json:"status"`
	CodeType1   any        `json:"err-code"` // can be either a string or int depending on the endpoint
	ErrMsgType1 string     `json:"err-msg"`
	CodeType2   any        `json:"err_code"`
	ErrMsgType2 string     `json:"err_msg"`
	Timestamp   types.Time `json:"ts"`
}

// MarketSummary24HourResponse contains the rolling 24-hour market summary for one spot symbol
type MarketSummary24HourResponse struct {
	Tick MarketSummary24HourTick `json:"tick"`
}

// CurrencyChains stores currency and chain info
type CurrencyChains struct {
	Currency   string           `json:"currency"`
	AssetType  uint8            `json:"assetType"`
	InstStatus string           `json:"instStatus"`
	Chains     []*CurrencyChain `json:"chains"`
}

// WebsocketFuturesKline stores kline data for futures and swap websocket
type WebsocketFuturesKline struct {
	Channel   string                    `json:"ch"`
	Timestamp types.Time                `json:"ts"`
	Tick      WebsocketFuturesKlineTick `json:"tick"`
}

// WebsocketMarketDepth stores market depth data for futures and swap websocket
type WebsocketMarketDepth struct {
	Channel   string                   `json:"ch"`
	Timestamp types.Time               `json:"ts"`
	Tick      WebsocketMarketDepthTick `json:"tick"`
}

// WebsocketIncrementalMarketDepth stores incremental market depth data for swap and futures websocket
type WebsocketIncrementalMarketDepth struct {
	Channel   string                              `json:"ch"`
	Timestamp types.Time                          `json:"ts"`
	Tick      WebsocketIncrementalMarketDepthTick `json:"tick"`
}

// WebsocketMarketDetail stores market detail data for futures and swap websocket
type WebsocketMarketDetail struct {
	Channel   string                    `json:"ch"`
	Timestamp types.Time                `json:"ts"`
	Tick      WebsocketMarketDetailTick `json:"tick"`
}

// WebsocketMarketBBO stores BBO data for futures and swap websocket
type WebsocketMarketBBO struct {
	Channel   string                 `json:"ch"`
	Timestamp types.Time             `json:"ts"`
	Tick      WebsocketMarketBBOTick `json:"tick"`
}

// WebsocketSubTradeDetail stores trade detail data for futures websocket
type WebsocketSubTradeDetail struct {
	Channel   string                      `json:"ch"`
	Timestamp types.Time                  `json:"ts"`
	Tick      WebsocketSubTradeDetailTick `json:"tick"`
}

// --------------------------------Spot-----------------------------------------

// Response stores the HTX response information
type Response struct {
	Status       string     `json:"status"`
	Channel      string     `json:"ch"`
	Timestamp    types.Time `json:"ts"`
	ErrorCode    string     `json:"err-code"`
	ErrorMessage string     `json:"err-msg"`
}

// MarginRatesResponse stores margin rates data
type MarginRatesResponse struct {
	Data []MarginRate `json:"data"`
}

// ResponseV2 stores the HTX generic response info
type ResponseV2 struct {
	Code    int32  `json:"code"`
	Message string `json:"message"`
}

// SwapMarket stores market data for swaps
type SwapMarket struct {
	Symbol         string     `json:"symbol"`
	ContractCode   string     `json:"contract_code"`
	ContractSize   float64    `json:"contract_size"`
	PriceTick      float64    `json:"price_tick"`
	SettlementDate types.Time `json:"settlement_date"`
	CreateDate     types.Time `json:"create_date"`
	DeliveryTime   types.Time `json:"delivery_time"`
	ContractStatus uint64     `json:"contract_status"`
}

// KlineItem stores a kline item
type KlineItem struct {
	IDTimestamp types.Time `json:"id"`
	Open        float64    `json:"open"`
	Close       float64    `json:"close"`
	Low         float64    `json:"low"`
	High        float64    `json:"high"`
	Amount      float64    `json:"amount"`
	Volume      float64    `json:"vol"`
	Count       uint64     `json:"count"`
}

// CancelOpenOrdersBatchResponse retains cancellation counts and any remaining-order cursor.
type CancelOpenOrdersBatchResponse struct {
	Data         OpenOrderCancellationResult `json:"data"`
	Status       string                      `json:"status"`
	ErrorMessage string                      `json:"err-msg"`
}

// DetailMerged stores the ticker detail merged data
type DetailMerged struct {
	Detail
	Version uint64    `json:"version"`
	Ask     []float64 `json:"ask"`
	Bid     []float64 `json:"bid"`
}

// TickersResponse contains the rolling summaries for all spot symbols.
type TickersResponse struct {
	Data []Ticker `json:"data"`
}

// FuturesBatchTicker holds ticker data
type FuturesBatchTicker struct {
	ID             float64      `json:"id"`
	Timestamp      types.Time   `json:"ts"`
	Ask            []float64    `json:"ask"`
	Bid            []float64    `json:"bid"`
	BusinessType   string       `json:"business_type"`
	ContractCode   string       `json:"contract_code"`
	Open           types.Number `json:"open"`
	Close          types.Number `json:"close"`
	Low            types.Number `json:"low"`
	High           types.Number `json:"high"`
	Amount         types.Number `json:"amount"`
	Count          float64      `json:"count"`
	Volume         types.Number `json:"vol"`
	TradeTurnover  types.Number `json:"trade_turnover"`
	TradePartition string       `json:"trade_partition"`
	Symbol         string       `json:"symbol"` // If ContractCode is empty, Symbol is populated
}

// Ticker latest ticker data
type Ticker struct {
	Symbol  string  `json:"symbol"`
	Open    float64 `json:"open"`
	High    float64 `json:"high"`
	Low     float64 `json:"low"`
	Close   float64 `json:"close"`
	Amount  float64 `json:"amount"`
	Volume  float64 `json:"vol"`
	Count   float64 `json:"count"`
	Bid     float64 `json:"bid"`
	BidSize float64 `json:"bidSize"`
	Ask     float64 `json:"ask"`
	AskSize float64 `json:"askSize"`
}

// OrderBookDepthType var for request param types
type OrderBookDepthType string

// OrderBookDepthType values select the market depth aggregation level.
var (
	OrderBookDepthTypeNone  = OrderBookDepthType("")
	OrderBookDepthTypeStep0 = OrderBookDepthType("step0")
	OrderBookDepthTypeStep1 = OrderBookDepthType("step1")
	OrderBookDepthTypeStep2 = OrderBookDepthType("step2")
	OrderBookDepthTypeStep3 = OrderBookDepthType("step3")
	OrderBookDepthTypeStep4 = OrderBookDepthType("step4")
	OrderBookDepthTypeStep5 = OrderBookDepthType("step5")
)

// OrderBookRequest selects the market and order book aggregation level.
type OrderBookRequest struct {
	Symbol currency.Pair      // Required; example LTCBTC,BTCUSDT
	Type   OrderBookDepthType // step0, step1, step2, step3, step4, step5 (combined depth 0-5); when step0, no depth is merged
}

// Orderbook stores the orderbook data
type Orderbook struct {
	ID         uint64       `json:"id"`
	Timetstamp types.Time   `json:"ts"`
	Bids       [][2]float64 `json:"bids"`
	Asks       [][2]float64 `json:"asks"`
}

// Trade stores the trade data
type Trade struct {
	TradeID   float64    `json:"trade-id"`
	Price     float64    `json:"price"`
	Amount    float64    `json:"amount"`
	Direction string     `json:"direction"`
	Timestamp types.Time `json:"ts"`
}

// TradeHistory stores the trade history data
type TradeHistory struct {
	ID        uint64     `json:"id"`
	Timestamp types.Time `json:"ts"`
	Trades    []Trade    `json:"data"`
}

// Detail stores the ticker detail data
type Detail struct {
	Amount    float64    `json:"amount"`
	Open      float64    `json:"open"`
	Close     float64    `json:"close"`
	High      float64    `json:"high"`
	Timestamp types.Time `json:"timestamp"`
	ID        uint64     `json:"id"`
	Count     uint64     `json:"count"`
	Low       float64    `json:"low"`
	Volume    float64    `json:"vol"`
}

// Symbol stores the symbol data
type Symbol struct {
	BaseCurrency             string  `json:"base-currency"`
	QuoteCurrency            string  `json:"quote-currency"`
	PricePrecision           float64 `json:"price-precision"`
	AmountPrecision          float64 `json:"amount-precision"`
	SymbolPartition          string  `json:"symbol-partition"`
	Symbol                   string  `json:"symbol"`
	State                    string  `json:"state"`
	ValuePrecision           float64 `json:"value-precision"`
	MinOrderAmt              float64 `json:"min-order-amt"`
	MaxOrderAmt              float64 `json:"max-order-amt"`
	MinOrderValue            float64 `json:"min-order-value"`
	LimitOrderMinOrderAmt    float64 `json:"limit-order-min-order-amt"`
	LimitOrderMaxOrderAmt    float64 `json:"limit-order-max-order-amt"`
	SellMarketMinOrderAmt    float64 `json:"sell-market-min-order-amt"`
	SellMarketMaxOrderAmt    float64 `json:"sell-market-max-order-amt"`
	BuyMarketMaxOrderAmt     float64 `json:"buy-market-max-order-amt"`
	LeverageRatio            float64 `json:"leverage-ratio"`
	SuperMarginLeverageRatio float64 `json:"super-margin-leverage-ratio"`
	FundingLeverageRatio     float64 `json:"funding-leverage-ratio"`
}

// Account stores the account data
type Account struct {
	ID     uint64 `json:"id"`
	Type   string `json:"type"`
	State  string `json:"state"`
	UserID uint64 `json:"user-id"`
}

// AccountBalance stores the user all account balance
type AccountBalance struct {
	ID                    uint64                 `json:"id"`
	Type                  string                 `json:"type"`
	State                 string                 `json:"state"`
	AccountBalanceDetails []AccountBalanceDetail `json:"list"`
}

// AccountBalanceDetail stores the user account balance
type AccountBalanceDetail struct {
	Currency currency.Code `json:"currency"`
	Type     string        `json:"type"`
	Balance  types.Number  `json:"balance"`
}

// AggregatedBalance stores balances of all the sub-account
type AggregatedBalance struct {
	Currency string       `json:"currency"`
	Balance  types.Number `json:"balance"`
}

// CancelOrderBatch stores the cancel order batch data
type CancelOrderBatch struct {
	Success []string                   `json:"success"`
	Failed  []OrderCancellationFailure `json:"failed"`
}

// OrderInfo stores the order info
type OrderInfo struct {
	ID               uint64       `json:"id"`
	Symbol           string       `json:"symbol"`
	AccountID        uint64       `json:"account-id"`
	Amount           types.Number `json:"amount"`
	Price            types.Number `json:"price"`
	CreatedAt        types.Time   `json:"created-at"`
	Type             string       `json:"type"`
	FieldFees        types.Number `json:"field-fees"`
	FieldAmount      types.Number `json:"field-amount"`
	FieldCashAmount  types.Number `json:"field-cash-amount"`
	FilledAmount     types.Number `json:"filled-amount"`
	FilledCashAmount types.Number `json:"filled-cash-amount"`
	FilledFees       types.Number `json:"filled-fees"`
	FinishedAt       types.Time   `json:"finished-at"`
	UserID           uint64       `json:"user-id"`
	Source           string       `json:"source"`
	State            string       `json:"state"`
	CanceledAt       types.Time   `json:"canceled-at"`
	Exchange         string       `json:"exchange"`
	Batch            string       `json:"batch"`
}

// OrderMatchInfo stores the order match info
type OrderMatchInfo struct {
	ID           uint64     `json:"id"`
	OrderID      uint64     `json:"order-id"`
	MatchID      uint64     `json:"match-id"`
	Symbol       string     `json:"symbol"`
	Type         string     `json:"type"`
	Source       string     `json:"source"`
	Price        string     `json:"price"`
	FilledAmount string     `json:"filled-amount"`
	FilledFees   string     `json:"filled-fees"`
	CreatedAt    types.Time `json:"created-at"`
}

// MarginOrder stores the margin order info
type MarginOrder struct {
	Currency        string     `json:"currency"`
	Symbol          string     `json:"symbol"`
	AccruedAt       types.Time `json:"accrued-at"`
	LoanAmount      string     `json:"loan-amount"`
	LoanBalance     string     `json:"loan-balance"`
	InterestBalance string     `json:"interest-balance"`
	CreatedAt       types.Time `json:"created-at"`
	InterestAmount  string     `json:"interest-amount"`
	InterestRate    string     `json:"interest-rate"`
	AccountID       uint64     `json:"account-id"`
	UserID          uint64     `json:"user-id"`
	UpdatedAt       types.Time `json:"updated-at"`
	ID              uint64     `json:"id"`
	State           string     `json:"state"`
}

// MarginAccountBalance stores the margin account balance info
type MarginAccountBalance struct {
	ID       uint64           `json:"id"`
	Type     string           `json:"type"`
	State    string           `json:"state"`
	Symbol   string           `json:"symbol"`
	FlPrice  string           `json:"fl-price"`
	FlType   string           `json:"fl-type"`
	RiskRate string           `json:"risk-rate"`
	List     []AccountBalance `json:"list"`
}

// SpotOrderRequest holds the params required to place an order
type SpotOrderRequest struct {
	AccountID uint64 `json:"account-id,string"` // Account ID, obtained using the accounts method. Currency trades use the accountid of the ‘spot’ account; for loan asset transactions, please use the accountid of the ‘margin’ account.
	// ClientOrderID lets callers reconcile an accepted order when the HTTP response is lost.
	ClientOrderID string        `json:"client-order-id,omitempty"`
	Amount        float64       `json:"amount"` // The limit price indicates the quantity of the order, the market price indicates how much to buy when the order is paid, and the market price indicates how much the coin is sold when the order is sold.
	Price         float64       `json:"price"`  // Order price, market price does not use  this parameter
	Source        string        `json:"source"` // Order source, api: API call, margin-api: loan asset transaction
	Symbol        currency.Pair `json:"symbol"` // The symbol to use; example btcusdt, bccbtc......
	Type          SpotOrderType `json:"type"`   // 订单类型, buy-market: 市价买, sell-market: 市价卖, buy-limit: 限价买, sell-limit: 限价卖
}

// DepositAddress stores the users deposit address info
type DepositAddress struct {
	UserID     uint64 `json:"userId"`
	Currency   string `json:"currency"`
	Address    string `json:"address"`
	AddressTag string `json:"addressTag"`
	Chain      string `json:"chain"`
}

// ChainQuota stores the users currency chain quota
type ChainQuota struct {
	Chain                         string       `json:"chain"`
	MaxWithdrawAmount             types.Number `json:"maxWithdrawAmt"`
	WithdrawQuotaPerDay           types.Number `json:"withdrawQuotaPerDay"`
	RemainingWithdrawQuotaPerDay  types.Number `json:"remainWithdrawQuotaPerDay"`
	WithdrawQuotaPerYear          types.Number `json:"withdrawQuotaPerYear"`
	RemainingWithdrawQuotaPerYear types.Number `json:"remainWithdrawQuotaPerYear"`
	WithdrawQuotaTotal            types.Number `json:"withdrawQuotaTotal"`
	RemainingWithdrawQuotaTotal   types.Number `json:"remainWithdrawQuotaTotal"`
}

// WithdrawQuota stores the users withdraw quotas
type WithdrawQuota struct {
	Currency string       `json:"currency"`
	Chains   []ChainQuota `json:"chains"`
}

// SpotOrderType order type
type SpotOrderType string

var (
	// SpotOrderTypeBuyMarket buy market order
	SpotOrderTypeBuyMarket = SpotOrderType("buy-market")

	// SpotOrderTypeSellMarket sell market order
	SpotOrderTypeSellMarket = SpotOrderType("sell-market")

	// SpotOrderTypeBuyLimit buy limit order
	SpotOrderTypeBuyLimit = SpotOrderType("buy-limit")

	// SpotOrderTypeSellLimit sell limit order
	SpotOrderTypeSellLimit = SpotOrderType("sell-limit")
)

//-----------

// KlinesRequest represents Klines request data.
type KlinesRequest struct {
	Symbol currency.Pair // Symbol to be used; example btcusdt, bccbtc......
	Period string        // Kline time interval; 1min, 5min, 15min......
	Size   uint64        // Size; [1-2000]
}

// OrderVars stores side, status and type for any order/trade
type OrderVars struct {
	Side        order.Side
	Status      order.Status
	OrderType   order.Type
	TimeInForce order.TimeInForce
	Fee         float64
}

// Variables below are used to check api requests being sent out

var (
	validPeriods = []string{"1min", "5min", "15min", "30min", "60min", "4hour", "1day", "1mon"}

	validBasisPriceTypes = []string{"open", "close", "high", "low", "average"}

	validAmountType = map[string]uint64{
		"cont":           1,
		"cryptocurrency": 2,
	}

	validTransferType = []string{
		"master_to_sub", "sub_to_master",
	}

	validTradeTypes = map[string]uint64{
		"filled": 0,
		"closed": 5,
		"open":   6,
	}

	validOrderType = map[string]uint64{
		"quotation":         1,
		"cancelledOrder":    2,
		"forcedLiquidation": 3,
		"deliveryOrder":     4,
	}

	validOrderTypes = []string{
		orderPriceTypePostOnly,
		orderPriceTypeLimit, orderPriceTypeOpponent, orderPriceTypeLightning, orderPriceTypeOptimal5, orderPriceTypeOptimal10, orderPriceTypeOptimal20,
		orderTimeInForceFOK, orderTimeInForceIOC, "opponent_ioc", orderPriceTypeLightningIOC, "optimal_5_ioc",
		"optimal_10_ioc", orderPriceTypeOptimal20IOC, "opponent_fok", orderPriceTypeOptimal20FOK,
	}

	validTriggerType = map[string]string{
		"greaterOrEqual": "ge",
		"smallerOrEqual": "le",
	}

	validOrderPriceType = []string{
		orderPriceTypeLimit, orderPriceTypeOptimal5, orderPriceTypeOptimal10, orderPriceTypeOptimal20,
	}

	validLightningOrderPriceType = []string{
		orderPriceTypeLightning, "lightning_fok", orderPriceTypeLightningIOC,
	}

	validTradeType = map[string]uint64{
		"all":            0,
		"openLong":       1,
		"openShort":      2,
		"closeShort":     3,
		"closeLong":      4,
		"liquidateLong":  5,
		"liquidateShort": 6,
	}

	validFuturesTradeType = map[string]uint64{
		"all":            0,
		"openLong":       1,
		"openShort":      2,
		"closeShort":     3,
		"closeLong":      4,
		"liquidateLong":  5,
		"liquidateShort": 6,
		"deliveryLong":   7,
		"deliveryShort":  8,
		"reduceLong":     11,
		"reduceShort":    12,
	}

	contractExpiryNames = map[string]string{
		"this_week":    "CW",
		"next_week":    "NW",
		"quarter":      "CQ",
		"next_quarter": "NQ",
	}

	validContractExpiryCodes = []string{"CW", "NW", "CQ", "NQ"}

	validFuturesPeriods = []string{
		"1min", "5min", "15min", "30min", "60min", "1hour", "4hour", "1day",
	}

	validFuturesOrderPriceTypes = []string{
		orderPriceTypePostOnly,
		orderPriceTypeLimit, orderPriceTypeOpponent, orderPriceTypeLightning, orderPriceTypeOptimal5, orderPriceTypeOptimal10,
		orderPriceTypeOptimal20, orderTimeInForceFOK, orderTimeInForceIOC, "opponent_ioc", orderPriceTypeLightningIOC,
		"optimal_5_ioc", "optimal_10_ioc", orderPriceTypeOptimal20IOC, "opponent_fok",
		"lightning_fok", "optimal_5_fok", "optimal_10_fok", orderPriceTypeOptimal20FOK,
	}

	validFuturesRecordTypes = map[string]string{
		"closeLong":                   "3",
		"closeShort":                  "4",
		"openOpenPositionsTakerFees":  "5",
		"openPositionsMakerFees":      "6",
		"closePositionsTakerFees":     "7",
		"closePositionsMakerFees":     "8",
		"closeLongDelivery":           "9",
		"closeShortDelivery":          "10",
		"deliveryFee":                 "11",
		"longLiquidationClose":        "12",
		"shortLiquidationClose":       "13",
		"transferFromSpotToContracts": "14",
		"transferFromContractsToSpot": "15",
		"settleUnrealizedLongPNL":     "16",
		"settleUnrealizedShortPNL":    "17",
		"clawback":                    "19",
		"system":                      "26",
		"activityPrizeRewards":        "28",
		"rebate":                      "29",
		"transferToSub":               "34",
		"transferFromSub":             "35",
		"transferToMaster":            "36",
		"transferFromMaster":          "37",
	}

	validOffsetTypes = []string{
		"open", "close",
	}

	validOPTypes = []string{
		orderPriceTypeLightning, "lightning_fok", orderPriceTypeLightningIOC,
	}

	validFuturesReqType = map[string]uint64{
		"all":            1,
		"finishedStatus": 2,
	}

	validFuturesOrderTypes = map[string]uint64{
		orderPriceTypeLimit:     1,
		orderPriceTypeOpponent:  3,
		orderPriceTypeLightning: 4,
		"triggerOrder":          5,
		"postOnly":              6,
		orderPriceTypeOptimal5:  7,
		orderPriceTypeOptimal10: 8,
		orderPriceTypeOptimal20: 9,
		orderTimeInForceFOK:     10,
		orderTimeInForceIOC:     11,
	}

	validOrderStatus = map[order.Status]uint64{
		order.AnyStatus:          0,
		order.Active:             3,
		order.PartiallyFilled:    4,
		order.PartiallyCancelled: 5,
		order.Filled:             6,
		order.Cancelled:          7,
	}

	validStatusTypes = map[string]uint64{
		"all":       0,
		"success":   4,
		"failed":    5,
		"cancelled": 6,
	}
)

// WithdrawalHistoryResponse lists deposits or withdrawals matching the requested filters.
type WithdrawalHistoryResponse struct {
	Status string       `json:"status"`
	Data   []Withdrawal `json:"data"`
}

// Withdrawal contains details of a withdrawal
type Withdrawal struct {
	ID              uint64        `json:"id"`
	Type            string        `json:"type"`
	Currency        currency.Code `json:"currency"`
	TransactionHash string        `json:"tx-hash"`
	Chain           string        `json:"chain"`
	Amount          float64       `json:"amount"`
	SubType         string        `json:"sub-type"`
	Address         string        `json:"address"`
	AddressTag      string        `json:"address-tag"`
	FromAddressTag  string        `json:"from-addr-tag"`
	Fee             float64       `json:"fee"`
	State           string        `json:"state"`
	ErrorCode       string        `json:"error-code"`
	ErrorMessage    string        `json:"error-msg"`
	CreatedAt       types.Time    `json:"created-at"`
	UpdatedAt       types.Time    `json:"updated-at"`
}

func isEmptyHTXData(data []byte) bool {
	trimmed := bytes.TrimSpace(data)
	return len(trimmed) == 0 ||
		bytes.Equal(trimmed, []byte(`""`)) ||
		bytes.Equal(trimmed, []byte("null"))
}

// SpotKlineResponse contains spot candles ordered as supplied by the history endpoint.
type SpotKlineResponse struct {
	Response
	Data []KlineItem `json:"data"`
}

// MarketDetailMergedResponse wraps a rolling spot summary and the best bid and ask.
type MarketDetailMergedResponse struct {
	Response
	Tick DetailMerged `json:"tick"`
}

// OrderBookResponse wraps a depth snapshot and the exchange response status.
type OrderBookResponse struct {
	Response
	Depth Orderbook `json:"tick"`
}

// TradesResponse groups the latest executions under one exchange update.
type TradesResponse struct {
	Response
	Tick TradesTick `json:"tick"`
}

// TradeHistoryResponse contains batches of recent spot executions.
type TradeHistoryResponse struct {
	Response
	TradeHistory []TradeHistory `json:"data"`
}

// MarketDetailResponse wraps the rolling spot summary for one symbol.
type MarketDetailResponse struct {
	Response
	Tick Detail `json:"tick"`
}

// SymbolsResponse lists the spot symbols and their trading rules.
type SymbolsResponse struct {
	Response
	Symbols []Symbol `json:"data"`
}

// CurrenciesResponse lists the currency codes recognised by the spot API.
type CurrenciesResponse struct {
	Response
	Currencies []string `json:"data"`
}

// FuturesBatchTickersResponse uses ticks for the list of derivative market summaries.
type FuturesBatchTickersResponse struct {
	Data []FuturesBatchTicker `json:"ticks"`
}

// CurrenciesIncludingChainsResponse lists supported networks and transfer rules for each currency.
type CurrenciesIncludingChainsResponse struct {
	Data []CurrencyChains `json:"data"`
}

// CurrentServerTimeResponse places the server Unix timestamp in data rather than ts.
type CurrentServerTimeResponse struct {
	Response
	Timestamp types.Time `json:"data"`
}

// AccountsResponse lists the user accounts that can be selected for spot requests.
type AccountsResponse struct {
	Accounts []Account `json:"data"`
}

// AccountBalanceResponse groups the balances belonging to the requested account.
type AccountBalanceResponse struct {
	AccountBalanceData AccountBalance `json:"data"`
}

// AggregatedBalanceResponse groups balances across the requested subaccounts.
type AggregatedBalanceResponse struct {
	AggregatedBalances []AggregatedBalance `json:"data"`
}

// spotOrderSubmissionRequest serialises spot order quantities and identifiers using HTX wire formats.
type spotOrderSubmissionRequest struct {
	AccountID     uint64 `json:"account-id,string"`
	ClientOrderID string `json:"client-order-id,omitempty"`
	Amount        string `json:"amount"`
	Price         string `json:"price"`
	Source        string `json:"source"`
	Symbol        string `json:"symbol"`
	Type          string `json:"type"`
}

// SpotOrderIDResponse decodes the quoted order ID returned by spot placement and cancellation.
type SpotOrderIDResponse struct {
	OrderID uint64 `json:"data,string"`
}

// cancelOpenOrdersBatchRequest scopes cancellation to a spot account and symbol.
type cancelOpenOrdersBatchRequest struct {
	AccountID string `json:"account-id"`
	Symbol    string `json:"symbol"`
}

// OrderResponse wraps a spot order with its cumulative execution totals.
type OrderResponse struct {
	Order OrderInfo `json:"data"`
}

// OrderMatchResultsResponse contains the individual fills belonging to a spot order.
type OrderMatchResultsResponse struct {
	Orders []OrderMatchInfo `json:"data"`
}

// OrdersResponse contains spot orders matching the supplied filters.
type OrdersResponse struct {
	Orders []OrderInfo `json:"data"`
}

// marginAmountRequest specifies an amount in the borrowing or transfer currency.
type marginAmountRequest struct {
	Symbol   string `json:"symbol"`
	Currency string `json:"currency"`
	Amount   string `json:"amount"`
}

// MarginTransferResponse identifies an accepted transfer to or from a margin account.
type MarginTransferResponse struct {
	TransferID uint64 `json:"data"`
}

// MarginOrderResponse identifies a newly accepted margin loan.
type MarginOrderResponse struct {
	MarginOrderID uint64 `json:"data"`
}

// marginRepaymentRequest serialises the loan repayment amount as a decimal string.
type marginRepaymentRequest struct {
	Amount string `json:"amount"`
}

// MarginLoanOrdersResponse lists margin loans matching the supplied filters.
type MarginLoanOrdersResponse struct {
	MarginLoanOrders []MarginOrder `json:"data"`
}

// MarginAccountBalanceResponse groups balances and borrowing liabilities by margin account.
type MarginAccountBalanceResponse struct {
	Balances []MarginAccountBalance `json:"data"`
}

// WithdrawResponse identifies the withdrawal accepted or cancelled by HTX.
type WithdrawResponse struct {
	WithdrawID uint64 `json:"data"`
}

// withdrawalRequest selects a withdrawal network and optional destination tag.
type withdrawalRequest struct {
	Address  string `json:"address"`
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
	Fee      string `json:"fee,omitempty"`
	Chain    string `json:"chain,omitempty"`
	AddrTag  string `json:"addr-tag,omitempty"`
}

// DepositAddressResponse lists deposit addresses for the requested currency.
type DepositAddressResponse struct {
	DepositAddress []DepositAddress `json:"data"`
}

// WithdrawQuotasResponse reports the remaining withdrawal allowance for one currency.
type WithdrawQuotasResponse struct {
	WithdrawQuota WithdrawQuota `json:"data"`
}

// MarketSummary24HourTick is the rolling 24-hour market summary returned by the spot detail endpoint.
type MarketSummary24HourTick struct {
	Amount  float64 `json:"amount"`
	Open    float64 `json:"open"`
	Close   float64 `json:"close"`
	High    float64 `json:"high"`
	ID      uint64  `json:"id"`
	Count   float64 `json:"count"`
	Low     float64 `json:"low"`
	Version uint64  `json:"version"`
	Volume  float64 `json:"vol"`
}

// CurrencyChain describes deposits, withdrawals and confirmation requirements for one supported network.
type CurrencyChain struct {
	Chain                     string       `json:"chain"`
	DisplayName               string       `json:"displayName"`
	BaseChain                 string       `json:"baseChain"`
	BaseChainProtocol         string       `json:"baseChainProtocol"`
	IsDynamic                 bool         `json:"isDynamic"`
	NumberOfConfirmations     uint16       `json:"numOfConfirmations"`
	NumberOfFastConfirmations uint16       `json:"numOfFastConfirmations"`
	DepositStatus             string       `json:"depositStatus"`
	MinimumDepositAmount      types.Number `json:"minDepositAmt"`
	WithdrawStatus            string       `json:"withdrawStatus"`
	MinimumWithdrawalAmount   types.Number `json:"minWithdrawAmt"`
	WithdrawPrecision         uint64       `json:"withdrawPrecision"`
	MaximumWithdrawAmount     types.Number `json:"maxWithdrawAmt"`
	WithdrawQuotaPerDay       types.Number `json:"withdrawQuotaPerDay"`
	WithdrawQuotaPerYear      types.Number `json:"withdrawQuotaPerYear"`
	WithdrawQuotaTotal        types.Number `json:"withdrawQuotaTotal"`
	WithdrawFeeType           string       `json:"withdrawFeeType"`
	TransactFeeWithdraw       types.Number `json:"transactFeeWithdraw"`
	AddressWithTag            bool         `json:"addrWithTag"`
	AddressDepositTag         bool         `json:"addrDepositTag"`
}

// WebsocketFuturesKlineTick is a delivery candle whose volume is measured in contracts and amount in coins.
type WebsocketFuturesKlineTick struct {
	ID     uint64  `json:"id"`
	MRID   uint64  `json:"mrid"`
	Volume float64 `json:"vol"`
	Count  float64 `json:"count"`
	Open   float64 `json:"open"`
	Close  float64 `json:"close"`
	Low    float64 `json:"low"`
	High   float64 `json:"high"`
	Amount float64 `json:"amount"`
}

// WebsocketMarketDepthTick is a complete depth snapshot; each level is a price and quantity pair.
type WebsocketMarketDepthTick struct {
	MRID      uint64       `json:"mrid"`
	ID        uint64       `json:"id"`
	Bids      [][2]float64 `json:"bids"`
	Asks      [][2]float64 `json:"asks"`
	Timestamp types.Time   `json:"ts"`
	Version   uint64       `json:"version"`
	Channel   string       `json:"ch"`
}

// WebsocketIncrementalMarketDepthTick carries depth changes; Event distinguishes snapshots from subsequent updates.
type WebsocketIncrementalMarketDepthTick struct {
	MRID      uint64       `json:"mrid"`
	ID        uint64       `json:"id"`
	Bids      [][2]float64 `json:"bids"`
	Asks      [][2]float64 `json:"asks"`
	Timestamp types.Time   `json:"ts"`
	Version   uint64       `json:"version"`
	Channel   string       `json:"ch"`
	Event     string       `json:"event"`
}

// WebsocketMarketDetailTick is the rolling market summary carried by a detail subscription.
type WebsocketMarketDetailTick struct {
	ID     uint64  `json:"id"`
	MRID   uint64  `json:"mrid"`
	Open   float64 `json:"open"`
	Close  float64 `json:"close"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Amount float64 `json:"amount"`
	Volume float64 `json:"vol"`
	Count  float64 `json:"count"`
}

// WebsocketMarketBBOTick carries the best bid and ask as price and quantity pairs.
type WebsocketMarketBBOTick struct {
	Channel   string     `json:"ch"`
	MRID      uint64     `json:"mrid"`
	ID        uint64     `json:"id"`
	Bid       [2]float64 `json:"bid"`
	Ask       [2]float64 `json:"ask"`
	Timestamp types.Time `json:"ts"`
	Version   uint64     `json:"version"`
}

// WebsocketSubTradeDetailTick groups executions published in one trade notification.
type WebsocketSubTradeDetailTick struct {
	ID        uint64                  `json:"id"`
	Timestamp types.Time              `json:"ts"`
	Data      []WebsocketFuturesTrade `json:"data"`
}

// MarginRate groups the borrowing currencies available for a margin trading symbol.
type MarginRate struct {
	Symbol     string               `json:"symbol"`
	Currencies []MarginRateCurrency `json:"currencies"`
}

// OpenOrderCancellationResult reports cancellation counts and a cursor for orders still awaiting cancellation.
type OpenOrderCancellationResult struct {
	FailedCount  uint64 `json:"failed-count"`
	NextID       int64  `json:"next-id"` // Signed because the API sends -1 when no open orders remain
	SuccessCount uint64 `json:"success-count"`
}

// OrderCancellationFailure identifies an individual rejected cancellation within an otherwise successful batch request.
type OrderCancellationFailure struct {
	OrderID       string `json:"order-id"`
	ClientOrderID string `json:"client-order-id"`
	ErrorCode     string `json:"err-code"`
	ErrorMessage  string `json:"err-msg"`
}

// TradesTick groups executions returned by the latest-trade endpoint.
type TradesTick struct {
	Data []Trade `json:"data"`
}

// WebsocketFuturesTrade is one execution with its taker direction and exchange timestamp.
type WebsocketFuturesTrade struct {
	Amount    float64    `json:"amount"`
	Timestamp types.Time `json:"ts"`
	ID        uint64     `json:"id"`
	Price     float64    `json:"price"`
	Direction string     `json:"direction"`
}

// MarginRateCurrency describes the borrowing limits and interest rates for one margin currency.
type MarginRateCurrency struct {
	Currency       string       `json:"currency"`
	InterestRate   types.Number `json:"interest-rate"`
	MinLoanAmount  types.Number `json:"min-loan-amt"`
	MaxLoanAmount  types.Number `json:"max-loan-amt"`
	LoanableAmount types.Number `json:"loanable-amt"`
	ActualRate     types.Number `json:"actual-rate"`
}

// CancelOrderBatchResponse contains the result of one batch cancellation request.
type CancelOrderBatchResponse struct {
	Response
	Data *CancelOrderBatch `json:"data"`
}

// cancelOrderBatchRequest accepts one identifier kind per request.
type cancelOrderBatchRequest struct {
	ClientOrderIDs []string `json:"client-order-ids,omitempty"`
	OrderIDs       []string `json:"order-ids,omitempty"`
}
