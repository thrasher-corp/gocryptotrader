package binance

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// OptionsAccountFundingFlowRequest holds the parameters for querying options account funding flows
type OptionsAccountFundingFlowRequest struct {
	Currency  currency.Code
	RecordID  uint64
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
}

// OptionsAccountFunding holds an options account funding flow
type OptionsAccountFunding struct {
	ID    uint64        `json:"id"`
	Asset currency.Code `json:"asset"`
	// Amount is positive for inflows and negative for outflows
	Amount     types.Number `json:"amount"`
	Type       string       `json:"type"`
	CreateDate types.Time   `json:"createDate"`
}

// OptionsMarginAccountInformationResponse holds the options margin account information
type OptionsMarginAccountInformationResponse struct {
	Asset       []OptionsMarginAccountAsset `json:"asset"`
	Greek       []OptionsGreek              `json:"greek"`
	Time        types.Time                  `json:"time"`
	CanTrade    bool                        `json:"canTrade"`
	CanDeposit  bool                        `json:"canDeposit"`
	CanWithdraw bool                        `json:"canWithdraw"`
	ReduceOnly  bool                        `json:"reduceOnly"`
	// TradeGroupID is -1 when the account is not in a trade group
	TradeGroupID int64 `json:"tradeGroupId"`
}

// OptionsMarginAccountAsset holds an options margin account asset balance
type OptionsMarginAccountAsset struct {
	Asset             currency.Code `json:"asset"`
	MarginBalance     types.Number  `json:"marginBalance"`
	Equity            types.Number  `json:"equity"`
	Available         types.Number  `json:"available"`
	InitialMargin     types.Number  `json:"initialMargin"`
	MaintenanceMargin types.Number  `json:"maintMargin"`
	UnrealizedPNL     types.Number  `json:"unrealizedPNL"`
	AdjustedEquity    types.Number  `json:"adjustedEquity"`
}

// OptionsGreek holds the options account greeks of an underlying
type OptionsGreek struct {
	Underlying string       `json:"underlying"`
	Delta      types.Number `json:"delta"`
	Gamma      types.Number `json:"gamma"`
	Theta      types.Number `json:"theta"`
	Vega       types.Number `json:"vega"`
}

// OptionsTransactionHistoryDownloadIDResponse holds the download ID of an options transaction history file
type OptionsTransactionHistoryDownloadIDResponse struct {
	// AverageCostTimestampOfLast30Days is the average time in milliseconds a download took over the last 30 days
	AverageCostTimestampOfLast30Days uint64 `json:"avgCostTimestampOfLast30d"`
	DownloadID                       string `json:"downloadId"`
}

// OptionsTransactionHistoryDownloadLinkResponse holds the download link of an options transaction history file
type OptionsTransactionHistoryDownloadLinkResponse struct {
	DownloadID string `json:"downloadId"`
	// Status is processing until the file is ready and completed after
	Status   string `json:"status"`
	URL      string `json:"url"`
	Notified bool   `json:"notified"`
	// ExpirationTimestamp is a Unix timestamp in milliseconds, or -1 while the file is processing
	ExpirationTimestamp int64 `json:"expirationTimestamp"`
	IsExpired           bool  `json:"isExpired"`
}

// OptionsServerTimeResponse holds the options server time
type OptionsServerTimeResponse struct {
	ServerTime types.Time `json:"serverTime"`
}

// OptionsExchangeInformationResponse holds the options trading rules and symbol information
type OptionsExchangeInformationResponse struct {
	Timezone        string                `json:"timezone"`
	ServerTime      types.Time            `json:"serverTime"`
	OptionContracts []OptionsContract     `json:"optionContracts"`
	OptionAssets    []OptionsAsset        `json:"optionAssets"`
	OptionSymbols   []OptionsSymbolDetail `json:"optionSymbols"`
	RateLimits      []RateLimitInfo       `json:"rateLimits"`
}

// OptionsContract holds an options contract underlying
type OptionsContract struct {
	BaseAsset   currency.Code `json:"baseAsset"`
	QuoteAsset  currency.Code `json:"quoteAsset"`
	Underlying  string        `json:"underlying"`
	SettleAsset currency.Code `json:"settleAsset"`
	// NakedSell is sent here but documented on the option symbols
	NakedSell bool `json:"nakedSell"`
}

// OptionsAsset holds an options asset
type OptionsAsset struct {
	Name currency.Code `json:"name"`
}

// OptionsSymbolDetail holds the trading rules of an option symbol
type OptionsSymbolDetail struct {
	ExpiryDate types.Time            `json:"expiryDate"`
	Filters    []OptionsSymbolFilter `json:"filters"`
	Symbol     string                `json:"symbol"`
	// Side is CALL or PUT
	Side        string       `json:"side"`
	StrikePrice types.Number `json:"strikePrice"`
	Underlying  string       `json:"underlying"`
	// Unit is the quantity of the underlying a contract represents
	Unit                 uint64        `json:"unit"`
	LiquidationFeeRate   types.Number  `json:"liquidationFeeRate"`
	MinQuantity          types.Number  `json:"minQty"`
	MaxQuantity          types.Number  `json:"maxQty"`
	InitialMargin        types.Number  `json:"initialMargin"`
	MaintenanceMargin    types.Number  `json:"maintenanceMargin"`
	MinInitialMargin     types.Number  `json:"minInitialMargin"`
	MinMaintenanceMargin types.Number  `json:"minMaintenanceMargin"`
	PriceScale           uint64        `json:"priceScale"`
	QuantityScale        uint64        `json:"quantityScale"`
	QuoteAsset           currency.Code `json:"quoteAsset"`
	ContractType         string        `json:"contractType"`
	UnderlyingType       string        `json:"underlyingType"`
	// NakedSell is documented here but sent on the option contracts
	NakedSell bool   `json:"nakedSell"`
	Status    string `json:"status"`
}

// OptionsSymbolFilter holds an option symbol filter; the fields set depend on FilterType
type OptionsSymbolFilter struct {
	FilterType  string       `json:"filterType"`
	MinPrice    types.Number `json:"minPrice"`
	MaxPrice    types.Number `json:"maxPrice"`
	TickSize    types.Number `json:"tickSize"`
	MinQuantity types.Number `json:"minQty"`
	MaxQuantity types.Number `json:"maxQty"`
	StepSize    types.Number `json:"stepSize"`
}

// OptionsExerciseHistoryRequest holds the parameters for querying options historical exercise records
type OptionsExerciseHistoryRequest struct {
	Underlying string
	StartTime  time.Time
	EndTime    time.Time
	Limit      uint64
}

// ExerciseHistoryItem holds an option's exercise record. StrikeResult is REALISTIC_VALUE_STRICKEN when the option was
// exercised and EXTRINSIC_VALUE_EXPIRED when it expired out of the money
type ExerciseHistoryItem struct {
	Symbol          string       `json:"symbol"`
	StrikePrice     types.Number `json:"strikePrice"`
	RealStrikePrice types.Number `json:"realStrikePrice"`
	ExpiryDate      types.Time   `json:"expiryDate"`
	StrikeResult    string       `json:"strikeResult"`
}

// OptionsIndexPriceResponse holds the spot index price of an options underlying
type OptionsIndexPriceResponse struct {
	Time       types.Time   `json:"time"`
	IndexPrice types.Number `json:"indexPrice"`
}

// OptionsKlineRequest holds the parameters for querying option klines
type OptionsKlineRequest struct {
	Symbol    currency.Pair
	Interval  kline.Interval
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
}

// EOptionsCandlestick holds an option kline
type EOptionsCandlestick struct {
	OpenTime                 types.Time
	Open                     types.Number
	High                     types.Number
	Low                      types.Number
	Close                    types.Number
	Volume                   types.Number
	CloseTime                types.Time
	QuoteAssetVolume         types.Number
	NumberOfTrades           uint64
	TakerBuyBaseAssetVolume  types.Number
	TakerBuyQuoteAssetVolume types.Number
}

// UnmarshalJSON decodes an option kline from its array form
func (c *EOptionsCandlestick) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[12]any{&c.OpenTime, &c.Open, &c.High, &c.Low, &c.Close, &c.Volume, &c.CloseTime, &c.QuoteAssetVolume, &c.NumberOfTrades, &c.TakerBuyBaseAssetVolume, &c.TakerBuyQuoteAssetVolume, nil})
}

// OptionsOpenInterest holds the open interest of an option
type OptionsOpenInterest struct {
	Symbol             string       `json:"symbol"`
	SumOpenInterest    types.Number `json:"sumOpenInterest"`
	SumOpenInterestUSD types.Number `json:"sumOpenInterestUsd"`
	// Timestamp is quoted on the wire
	Timestamp types.Time `json:"timestamp"`
}

// OptionMarkPrice holds an option's mark price and greeks
type OptionMarkPrice struct {
	Symbol    string       `json:"symbol"`
	MarkPrice types.Number `json:"markPrice"`
	// BidIV, AskIV and MarkIV are implied volatilities
	BidIV            types.Number `json:"bidIV"`
	AskIV            types.Number `json:"askIV"`
	MarkIV           types.Number `json:"markIV"`
	Delta            types.Number `json:"delta"`
	Theta            types.Number `json:"theta"`
	Gamma            types.Number `json:"gamma"`
	Vega             types.Number `json:"vega"`
	HighPriceLimit   types.Number `json:"highPriceLimit"`
	LowPriceLimit    types.Number `json:"lowPriceLimit"`
	RiskFreeInterest types.Number `json:"riskFreeInterest"`
}

// OptionsOrderBookResponse holds an option order book
type OptionsOrderBookResponse struct {
	Bids            orderbook.LevelsArrayPriceAmount `json:"bids"`
	Asks            orderbook.LevelsArrayPriceAmount `json:"asks"`
	TransactionTime types.Time                       `json:"T"`
	LastUpdateID    uint64                           `json:"lastUpdateId"`
}

// OptionsTrade holds an option market trade
type OptionsTrade struct {
	ID            uint64       `json:"id"`
	TradeID       uint64       `json:"tradeId"`
	Symbol        string       `json:"symbol"`
	Price         types.Number `json:"price"`
	Quantity      types.Number `json:"qty"`
	QuoteQuantity types.Number `json:"quoteQty"`
	// Side is the taker's direction: -1 sell, 1 buy
	Side int64      `json:"side"`
	Time types.Time `json:"time"`
}

// EOptionTicker holds an option's 24 hour price change statistics
type EOptionTicker struct {
	Symbol             string       `json:"symbol"`
	PriceChange        types.Number `json:"priceChange"`
	PriceChangePercent types.Number `json:"priceChangePercent"`
	LastPrice          types.Number `json:"lastPrice"`
	LastQuantity       types.Number `json:"lastQty"`
	Open               types.Number `json:"open"`
	High               types.Number `json:"high"`
	Low                types.Number `json:"low"`
	// Volume counts contracts and Amount is in the quote asset
	Volume   types.Number `json:"volume"`
	Amount   types.Number `json:"amount"`
	BidPrice types.Number `json:"bidPrice"`
	AskPrice types.Number `json:"askPrice"`
	// OpenTime and CloseTime are the times of the first and last trade in the window
	OpenTime     types.Time   `json:"openTime"`
	CloseTime    types.Time   `json:"closeTime"`
	FirstTradeID uint64       `json:"firstTradeId"`
	TradeCount   uint64       `json:"tradeCount"`
	StrikePrice  types.Number `json:"strikePrice"`
	// ExercisePrice is the estimated settlement price in the hour before exercise and the index price otherwise
	ExercisePrice types.Number `json:"exercisePrice"`
}

// OptionsAutoCancelAllOpenOrdersHeartbeatResponse holds the underlyings whose auto-cancel countdown a heartbeat reset
type OptionsAutoCancelAllOpenOrdersHeartbeatResponse struct {
	Underlyings []string `json:"underlyings"`
}

// OptionsAutoCancelAllOpenOrdersConfigResponse holds the auto-cancel countdown of an options underlying
type OptionsAutoCancelAllOpenOrdersConfigResponse struct {
	Underlying string `json:"underlying"`
	// CountdownTime is in milliseconds
	CountdownTime uint64 `json:"countdownTime"`
}

// OptionsMarketMakerProtectionResponse holds the market maker protection config of an options underlying
type OptionsMarketMakerProtectionResponse struct {
	UnderlyingID uint64 `json:"underlyingId"`
	Underlying   string `json:"underlying"`
	// WindowTimeInMilliseconds and FrozenTimeInMilliseconds are durations in milliseconds
	WindowTimeInMilliseconds uint64       `json:"windowTimeInMilliseconds"`
	FrozenTimeInMilliseconds uint64       `json:"frozenTimeInMilliseconds"`
	QuantityLimit            types.Number `json:"qtyLimit"`
	DeltaLimit               types.Number `json:"deltaLimit"`
	LastTriggerTime          types.Time   `json:"lastTriggerTime"`
}

// OptionsMarketMakerProtectionConfigRequest holds the parameters for setting the market maker protection config of an
// options underlying. A zero FrozenTime keeps protection triggered until it is reset
type OptionsMarketMakerProtectionConfigRequest struct {
	Underlying    string
	WindowTime    time.Duration
	FrozenTime    time.Duration
	QuantityLimit float64
	DeltaLimit    float64
}

// OptionsAccountTradeListRequest holds the parameters for querying options account trades
type OptionsAccountTradeListRequest struct {
	Symbol    currency.Pair
	FromID    uint64
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
}

// OptionsAccountTradeItem holds an options account trade
type OptionsAccountTradeItem struct {
	ID       uint64       `json:"id"`
	TradeID  uint64       `json:"tradeId"`
	OrderID  uint64       `json:"orderId"`
	Symbol   string       `json:"symbol"`
	Price    types.Number `json:"price"`
	Quantity types.Number `json:"quantity"`
	// Fee is negative when it was deducted
	Fee            types.Number  `json:"fee"`
	RealizedProfit types.Number  `json:"realizedProfit"`
	Side           string        `json:"side"`
	Type           string        `json:"type"`
	Liquidity      string        `json:"liquidity"`
	Time           types.Time    `json:"time"`
	PriceScale     uint64        `json:"priceScale"`
	QuantityScale  uint64        `json:"quantityScale"`
	OptionSide     string        `json:"optionSide"`
	QuoteAsset     currency.Code `json:"quoteAsset"`
}

// OptionsOrderRequest holds the parameters for placing an options order
type OptionsOrderRequest struct {
	Symbol                  currency.Pair
	Side                    string
	OrderType               string
	Quantity                float64
	Price                   float64
	TimeInForce             string
	ReduceOnly              bool
	PostOnly                bool
	NewOrderResponseType    string
	ClientOrderID           string
	IsMarketMakerProtection bool
	SelfTradePreventionMode string
}

// OptionsOrder holds the order fields every options order endpoint returns
type OptionsOrder struct {
	OrderID          uint64       `json:"orderId"`
	Symbol           string       `json:"symbol"`
	Price            types.Number `json:"price"`
	Quantity         types.Number `json:"quantity"`
	ExecutedQuantity types.Number `json:"executedQty"`
	Side             string       `json:"side"`
	Type             string       `json:"type"`
	TimeInForce      string       `json:"timeInForce"`
	ReduceOnly       bool         `json:"reduceOnly"`
	UpdateTime       types.Time   `json:"updateTime"`
	Status           string       `json:"status"`
	AveragePrice     types.Number `json:"avgPrice"`
	ClientOrderID    string       `json:"clientOrderId"`
	PriceScale       uint64       `json:"priceScale"`
	QuantityScale    uint64       `json:"quantityScale"`
	// OptionSide is CALL or PUT
	OptionSide            string        `json:"optionSide"`
	QuoteAsset            currency.Code `json:"quoteAsset"`
	MarketMakerProtection bool          `json:"mmp"`
}

// OptionsOrderResponse holds an options order as order placement returns it
type OptionsOrderResponse struct {
	OptionsOrder
	Fee                     types.Number `json:"fee"`
	PostOnly                bool         `json:"postOnly"`
	CreateTime              types.Time   `json:"createTime"`
	Source                  string       `json:"source"`
	SelfTradePreventionMode string       `json:"selfTradePreventionMode"`
}

// OptionsCancelledOrder holds an entry of an options batch cancellation
type OptionsCancelledOrder struct {
	OptionsOrder
	Fee                     types.Number `json:"fee"`
	CreateTime              types.Time   `json:"createTime"`
	Source                  string       `json:"source"`
	SelfTradePreventionMode string       `json:"selfTradePreventionMode"`
}

// OptionsOrderStatusResponse holds an options order as Query Single Order returns it
type OptionsOrderStatusResponse struct {
	OptionsOrder
	PostOnly                bool       `json:"postOnly"`
	CreateTime              types.Time `json:"createTime"`
	SelfTradePreventionMode string     `json:"selfTradePreventionMode"`
}

// OptionsCancelOrderResponse holds an options order as Cancel Option Order returns it
type OptionsCancelOrderResponse struct {
	OptionsOrder
	CreateDate              types.Time `json:"createDate"`
	Source                  string     `json:"source"`
	SelfTradePreventionMode string     `json:"selfTradePreventionMode"`
}

// OptionPosition holds an options position
type OptionPosition struct {
	EntryPrice types.Number `json:"entryPrice"`
	Symbol     string       `json:"symbol"`
	Side       string       `json:"side"`
	// Quantity is positive for long positions and negative for short positions
	Quantity      types.Number  `json:"quantity"`
	MarkValue     types.Number  `json:"markValue"`
	UnrealizedPNL types.Number  `json:"unrealizedPNL"`
	MarkPrice     types.Number  `json:"markPrice"`
	StrikePrice   types.Number  `json:"strikePrice"`
	ExpiryDate    types.Time    `json:"expiryDate"`
	PriceScale    uint64        `json:"priceScale"`
	QuantityScale uint64        `json:"quantityScale"`
	OptionSide    string        `json:"optionSide"`
	QuoteAsset    currency.Code `json:"quoteAsset"`
	Time          types.Time    `json:"time"`
	// BidQuantity and AskQuantity are the open buy and sell order quantities
	BidQuantity types.Number `json:"bidQuantity"`
	AskQuantity types.Number `json:"askQuantity"`
}

// OptionsOpenOrdersRequest holds the parameters for querying current open options orders
type OptionsOpenOrdersRequest struct {
	Symbol    currency.Pair
	OrderID   uint64
	StartTime time.Time
	EndTime   time.Time
}

// OptionsOpenOrder holds a current open options order
type OptionsOpenOrder struct {
	OptionsOrder
	CreateTime              types.Time `json:"createTime"`
	SelfTradePreventionMode string     `json:"selfTradePreventionMode"`
}

// OptionsOrderHistoryRequest holds the parameters for querying options order history
type OptionsOrderHistoryRequest struct {
	Symbol    currency.Pair
	OrderID   uint64
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
}

// OptionsOrderHistoryItem holds a finished options order
type OptionsOrderHistoryItem struct {
	OptionsOrder
	CreateTime types.Time `json:"createTime"`
}

// OptionsUserExerciseRecordRequest holds the parameters for querying the account's options exercise records
type OptionsUserExerciseRecordRequest struct {
	Symbol    currency.Pair
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
}

// UserOptionsExerciseRecord holds an options exercise record of the account
type UserOptionsExerciseRecord struct {
	// ID is quoted on the wire and exceeds float64 precision
	ID            string        `json:"id"`
	Currency      currency.Code `json:"currency"`
	Symbol        string        `json:"symbol"`
	ExercisePrice types.Number  `json:"exercisePrice"`
	Quantity      types.Number  `json:"quantity"`
	Amount        types.Number  `json:"amount"`
	Fee           types.Number  `json:"fee"`
	CreateDate    types.Time    `json:"createDate"`
	PriceScale    uint64        `json:"priceScale"`
	QuantityScale uint64        `json:"quantityScale"`
	OptionSide    string        `json:"optionSide"`
	PositionSide  string        `json:"positionSide"`
	QuoteAsset    currency.Code `json:"quoteAsset"`
}

// OptionsListenKeyResponse holds the listen key of an options user data stream and when it expires
type OptionsListenKeyResponse struct {
	ListenKey  string     `json:"listenKey"`
	Expiration types.Time `json:"expiration"`
}
