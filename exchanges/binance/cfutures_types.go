package binance

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// FuturesAccountInformationResponse holds the coin margined futures account information
type FuturesAccountInformationResponse struct {
	Assets      []FuturesAccountAsset               `json:"assets"`
	Positions   []FuturesAccountInformationPosition `json:"positions"`
	CanDeposit  bool                                `json:"canDeposit"`
	CanTrade    bool                                `json:"canTrade"`
	CanWithdraw bool                                `json:"canWithdraw"`
	FeeTier     uint64                              `json:"feeTier"`
	UpdateTime  types.Time                          `json:"updateTime"`
}

// FuturesAccountAsset holds a coin margined futures account asset balance
type FuturesAccountAsset struct {
	Asset                  currency.Code `json:"asset"`
	WalletBalance          types.Number  `json:"walletBalance"`
	UnrealizedProfit       types.Number  `json:"unrealizedProfit"`
	MarginBalance          types.Number  `json:"marginBalance"`
	MaintenanceMargin      types.Number  `json:"maintMargin"`
	InitialMargin          types.Number  `json:"initialMargin"`
	PositionInitialMargin  types.Number  `json:"positionInitialMargin"`
	OpenOrderInitialMargin types.Number  `json:"openOrderInitialMargin"`
	MaxWithdrawAmount      types.Number  `json:"maxWithdrawAmount"`
	CrossWalletBalance     types.Number  `json:"crossWalletBalance"`
	CrossUnPNL             types.Number  `json:"crossUnPnl"`
	AvailableBalance       types.Number  `json:"availableBalance"`
	UpdateTime             types.Time    `json:"updateTime"`
}

// FuturesAccountInformationPosition holds a coin margined futures account position
type FuturesAccountInformationPosition struct {
	Symbol                 string       `json:"symbol"`
	PositionAmount         types.Number `json:"positionAmt"`
	InitialMargin          types.Number `json:"initialMargin"`
	MaintenanceMargin      types.Number `json:"maintMargin"`
	UnrealizedProfit       types.Number `json:"unrealizedProfit"`
	PositionInitialMargin  types.Number `json:"positionInitialMargin"`
	OpenOrderInitialMargin types.Number `json:"openOrderInitialMargin"`
	Leverage               types.Number `json:"leverage"`
	Isolated               bool         `json:"isolated"`
	PositionSide           string       `json:"positionSide"`
	EntryPrice             types.Number `json:"entryPrice"`
	BreakEvenPrice         types.Number `json:"breakEvenPrice"`
	MaxQuantity            types.Number `json:"maxQty"`
	UpdateTime             types.Time   `json:"updateTime"`
	NotionalValue          types.Number `json:"notionalValue"`
	// IsolatedWallet is missing from the REST docs, but the WebSocket API's account.status schema for the same account
	// data documents it
	IsolatedWallet types.Number `json:"isolatedWallet"`
}

// FuturesAccountBalanceData holds a coin margined futures account balance
type FuturesAccountBalanceData struct {
	AccountAlias       string        `json:"accountAlias"`
	Asset              currency.Code `json:"asset"`
	Balance            types.Number  `json:"balance"`
	WithdrawAvailable  types.Number  `json:"withdrawAvailable"`
	CrossWalletBalance types.Number  `json:"crossWalletBalance"`
	CrossUnPNL         types.Number  `json:"crossUnPnl"`
	AvailableBalance   types.Number  `json:"availableBalance"`
	UpdateTime         types.Time    `json:"updateTime"`
}

// CFuturesIncomeHistoryRequest holds the parameters for querying coin margined futures income history
type CFuturesIncomeHistoryRequest struct {
	Symbol     currency.Pair
	IncomeType string
	StartTime  time.Time
	EndTime    time.Time
	Page       uint64
	Limit      uint64
}

// FuturesIncomeHistoryData holds a coin margined futures income record
type FuturesIncomeHistoryData struct {
	Symbol     string        `json:"symbol"`
	IncomeType string        `json:"incomeType"`
	Income     types.Number  `json:"income"`
	Asset      currency.Code `json:"asset"`
	Info       string        `json:"info"`
	Time       types.Time    `json:"time"`
	// TransactionID and TradeID are quoted on the wire and TradeID is empty for records that are not trades
	TransactionID string `json:"tranId"`
	TradeID       string `json:"tradeId"`
}

// NotionalBracketData holds the notional brackets of a coin margined futures pair
type NotionalBracketData struct {
	Pair     string            `json:"pair"`
	Brackets []NotionalBracket `json:"brackets"`
}

// NotionalBracket is a single leverage bracket
type NotionalBracket struct {
	Bracket                uint64  `json:"bracket"`
	InitialLeverage        uint64  `json:"initialLeverage"`
	QuantityCap            float64 `json:"qtyCap"`
	QuantityFloor          float64 `json:"qtylFloor"` // Binance's own spelling, typo included
	MaintenanceMarginRatio float64 `json:"maintMarginRatio"`
	Cumulative             float64 `json:"cum"`
}

// CFuturesStatisticsRequest holds the parameters shared by the coin margined futures statistics endpoints under
// /futures/data
type CFuturesStatisticsRequest struct {
	// Pair is the contract pair, such as BTCUSD
	Pair         currency.Code
	ContractType string
	Period       kline.Interval
	Limit        uint64
	StartTime    time.Time
	EndTime      time.Time
}

// FuturesBasisData holds a coin margined futures basis record
type FuturesBasisData struct {
	IndexPrice   types.Number `json:"indexPrice"`
	ContractType string       `json:"contractType"`
	BasisRate    types.Number `json:"basisRate"`
	FuturesPrice types.Number `json:"futuresPrice"`
	// AnnualizedBasisRate is empty for perpetual contracts
	AnnualizedBasisRate types.Number `json:"annualizedBasisRate"`
	Basis               types.Number `json:"basis"`
	Pair                string       `json:"pair"`
	Timestamp           types.Time   `json:"timestamp"`
}

// CFuturesAggregatedTradesRequest holds the parameters for querying coin margined futures aggregate trades
type CFuturesAggregatedTradesRequest struct {
	Symbol    currency.Pair
	FromID    uint64
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
}

// CFuturesAggregatedTrade holds a coin margined futures aggregate trade
type CFuturesAggregatedTrade struct {
	AggregateTradeID uint64       `json:"a"`
	Price            types.Number `json:"p"`
	Quantity         types.Number `json:"q"`
	FirstTradeID     uint64       `json:"f"`
	LastTradeID      uint64       `json:"l"`
	Timestamp        types.Time   `json:"T"`
	IsBuyerMaker     bool         `json:"m"`
}

// CFuturesContinuousKlineRequest holds the parameters for querying coin margined futures continuous contract klines
type CFuturesContinuousKlineRequest struct {
	// Pair is the contract pair, such as BTCUSD
	Pair         currency.Code
	ContractType string
	Interval     kline.Interval
	StartTime    time.Time
	EndTime      time.Time
	Limit        uint64
}

// CFuturesCandleStick holds a coin margined futures kline. Volume and TakerBuyVolume count contracts
type CFuturesCandleStick struct {
	OpenTime                types.Time
	Open                    types.Number
	High                    types.Number
	Low                     types.Number
	Close                   types.Number
	Volume                  types.Number
	CloseTime               types.Time
	BaseAssetVolume         types.Number
	NumberOfTrades          uint64
	TakerBuyVolume          types.Number
	TakerBuyBaseAssetVolume types.Number
}

// UnmarshalJSON decodes a coin margined futures kline from its array form
func (c *CFuturesCandleStick) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[12]any{&c.OpenTime, &c.Open, &c.High, &c.Low, &c.Close, &c.Volume, &c.CloseTime, &c.BaseAssetVolume, &c.NumberOfTrades, &c.TakerBuyVolume, &c.TakerBuyBaseAssetVolume, nil})
}

// CFuturesExchangeInfoResponse holds the coin margined futures trading rules and symbol information
type CFuturesExchangeInfoResponse struct {
	// ExchangeFilters is always empty in practice
	ExchangeFilters []string             `json:"exchangeFilters"`
	RateLimits      []RateLimitInfo      `json:"rateLimits"`
	ServerTime      types.Time           `json:"serverTime"`
	Symbols         []CFuturesSymbolInfo `json:"symbols"`
	Timezone        string               `json:"timezone"`
}

// CFuturesSymbolInfo holds the trading rules of a coin margined futures symbol
type CFuturesSymbolInfo struct {
	Filters        []CFuturesSymbolFilter `json:"filters"`
	OrderTypes     []string               `json:"orderTypes"`
	TimeInForce    []string               `json:"timeInForce"`
	LiquidationFee types.Number           `json:"liquidationFee"`
	// MarketTakeBound is the largest rate a market order's price may differ from the mark price
	MarketTakeBound        types.Number  `json:"marketTakeBound"`
	Symbol                 string        `json:"symbol"`
	Pair                   string        `json:"pair"`
	ContractType           string        `json:"contractType"`
	DeliveryDate           types.Time    `json:"deliveryDate"`
	OnboardDate            types.Time    `json:"onboardDate"`
	ContractStatus         string        `json:"contractStatus"`
	ContractSize           uint64        `json:"contractSize"`
	QuoteAsset             currency.Code `json:"quoteAsset"`
	BaseAsset              currency.Code `json:"baseAsset"`
	MarginAsset            currency.Code `json:"marginAsset"`
	PricePrecision         uint64        `json:"pricePrecision"`
	QuantityPrecision      uint64        `json:"quantityPrecision"`
	BaseAssetPrecision     uint64        `json:"baseAssetPrecision"`
	QuotePrecision         uint64        `json:"quotePrecision"`
	EqualQuantityPrecision uint64        `json:"equalQtyPrecision"`
	// TriggerProtect is the threshold for algo orders sent with priceProtect
	TriggerProtect           types.Number `json:"triggerProtect"`
	MaintenanceMarginPercent types.Number `json:"maintMarginPercent"`
	RequiredMarginPercent    types.Number `json:"requiredMarginPercent"`
	UnderlyingType           string       `json:"underlyingType"`
	UnderlyingSubType        []string     `json:"underlyingSubType"`
	// MaxMoveOrderLimit and PermissionSets are sent but not documented
	MaxMoveOrderLimit uint64   `json:"maxMoveOrderLimit"`
	PermissionSets    []string `json:"permissionSets"`
}

// CFuturesSymbolFilter holds a coin margined futures symbol filter; the fields set depend on FilterType
type CFuturesSymbolFilter struct {
	FilterType        string       `json:"filterType"`
	MaxPrice          types.Number `json:"maxPrice"`
	MinPrice          types.Number `json:"minPrice"`
	TickSize          types.Number `json:"tickSize"`
	MaxQuantity       types.Number `json:"maxQty"`
	MinQuantity       types.Number `json:"minQty"`
	StepSize          types.Number `json:"stepSize"`
	Limit             int64        `json:"limit"`
	MultiplierUp      types.Number `json:"multiplierUp"`
	MultiplierDown    types.Number `json:"multiplierDown"`
	MultiplierDecimal types.Number `json:"multiplierDecimal"`
}

// CFuturesFundingRateHistoryRequest holds the parameters for querying coin margined perpetual futures funding rates
type CFuturesFundingRateHistoryRequest struct {
	Symbol    currency.Pair
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
}

// CFuturesFundingRateHistory holds a coin margined perpetual futures funding rate
type CFuturesFundingRateHistory struct {
	Symbol      string       `json:"symbol"`
	FundingTime types.Time   `json:"fundingTime"`
	FundingRate types.Number `json:"fundingRate"`
	// MarkPrice and RateType are sent since the coin margined and USDⓈ-M futures integration but not documented
	MarkPrice types.Number `json:"markPrice"`
	RateType  string       `json:"rateType"`
}

// CFuturesFundingRateInfo holds the funding rate limits of a coin margined perpetual futures symbol whose cap, floor
// or funding interval was adjusted
type CFuturesFundingRateInfo struct {
	Symbol                   string       `json:"symbol"`
	AdjustedFundingRateCap   types.Number `json:"adjustedFundingRateCap"`
	AdjustedFundingRateFloor types.Number `json:"adjustedFundingRateFloor"`
	FundingIntervalHours     uint64       `json:"fundingIntervalHours"`
	Disclaimer               bool         `json:"disclaimer"`
}

// IndexMarkPrice holds the index and mark prices of a coin margined futures symbol
type IndexMarkPrice struct {
	Symbol     string       `json:"symbol"`
	Pair       string       `json:"pair"`
	MarkPrice  types.Number `json:"markPrice"`
	IndexPrice types.Number `json:"indexPrice"`
	// EstimatedSettlePrice is only useful in the last hour before settlement
	EstimatedSettlePrice types.Number `json:"estimatedSettlePrice"`
	// LastFundingRate, InterestRate and NextFundingTime are only set for perpetual contracts
	LastFundingRate types.Number `json:"lastFundingRate"`
	InterestRate    types.Number `json:"interestRate"`
	NextFundingTime types.Time   `json:"nextFundingTime"`
	Time            types.Time   `json:"time"`
}

// CFuturesIndexPriceKlineRequest holds the parameters for querying coin margined futures index price klines
type CFuturesIndexPriceKlineRequest struct {
	// Pair is the contract pair, such as BTCUSD
	Pair      currency.Code
	Interval  kline.Interval
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
}

// CFuturesPriceCandleStick holds a coin margined futures index or mark price kline
type CFuturesPriceCandleStick struct {
	OpenTime  types.Time
	Open      types.Number
	High      types.Number
	Low       types.Number
	Close     types.Number
	CloseTime types.Time
	// NumberOfBasicData counts the price samples the kline aggregates
	NumberOfBasicData uint64
}

// UnmarshalJSON decodes a coin margined futures index or mark price kline from its array form
func (c *CFuturesPriceCandleStick) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[12]any{&c.OpenTime, &c.Open, &c.High, &c.Low, &c.Close, nil, &c.CloseTime, nil, &c.NumberOfBasicData, nil, nil, nil})
}

// CFuturesKlineRequest holds the parameters for querying coin margined futures klines, mark price klines and premium
// index klines of a symbol
type CFuturesKlineRequest struct {
	Symbol    currency.Pair
	Interval  kline.Interval
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
}

// TopTraderAccountRatio holds a coin margined futures long/short account ratio, of the top traders or of every account
type TopTraderAccountRatio struct {
	Pair           string       `json:"pair"`
	LongShortRatio types.Number `json:"longShortRatio"`
	LongAccount    types.Number `json:"longAccount"`
	ShortAccount   types.Number `json:"shortAccount"`
	Timestamp      types.Time   `json:"timestamp"`
}

// FuturesPublicTradesData holds a coin margined futures trade. Quantity counts contracts and BaseQuantity is the
// traded base asset amount; coin margined futures sends no quote figure
type FuturesPublicTradesData struct {
	ID           uint64       `json:"id"`
	Price        types.Number `json:"price"`
	Quantity     types.Number `json:"qty"`
	BaseQuantity types.Number `json:"baseQty"`
	Time         types.Time   `json:"time"`
	IsBuyerMaker bool         `json:"isBuyerMaker"`
}

// CFuturesOpenInterestResponse holds the open interest of a coin margined futures symbol
type CFuturesOpenInterestResponse struct {
	Symbol       string       `json:"symbol"`
	Pair         string       `json:"pair"`
	OpenInterest types.Number `json:"openInterest"`
	ContractType string       `json:"contractType"`
	Time         types.Time   `json:"time"`
}

// OpenInterestStats holds a coin margined futures open interest statistic
type OpenInterestStats struct {
	Pair         string `json:"pair"`
	ContractType string `json:"contractType"`
	// SumOpenInterest counts contracts and SumOpenInterestValue is in the base asset
	SumOpenInterest      types.Number `json:"sumOpenInterest"`
	SumOpenInterestValue types.Number `json:"sumOpenInterestValue"`
	Timestamp            types.Time   `json:"timestamp"`
}

// CFuturesOrderBookResponse holds a coin margined futures order book
type CFuturesOrderBookResponse struct {
	LastUpdateID      uint64                           `json:"lastUpdateId"`
	Symbol            string                           `json:"symbol"`
	Pair              string                           `json:"pair"`
	MessageOutputTime types.Time                       `json:"E"`
	TransactionTime   types.Time                       `json:"T"`
	Bids              orderbook.LevelsArrayPriceAmount `json:"bids"`
	Asks              orderbook.LevelsArrayPriceAmount `json:"asks"`
}

// CFuturesPremiumIndexCandleStick holds a coin margined futures premium index kline
type CFuturesPremiumIndexCandleStick struct {
	OpenTime  types.Time
	Open      types.Number
	High      types.Number
	Low       types.Number
	Close     types.Number
	CloseTime types.Time
}

// UnmarshalJSON decodes a coin margined futures premium index kline from its array form
func (c *CFuturesPremiumIndexCandleStick) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[12]any{&c.OpenTime, &c.Open, &c.High, &c.Low, &c.Close, nil, &c.CloseTime, nil, nil, nil, nil, nil})
}

// CFuturesIndexPriceConstituentsResponse holds the constituents of a coin margined futures index price
type CFuturesIndexPriceConstituentsResponse struct {
	Symbol       string                          `json:"symbol"`
	Time         types.Time                      `json:"time"`
	Constituents []CFuturesIndexPriceConstituent `json:"constituents"`
}

// CFuturesIndexPriceConstituent holds an exchange price that makes up a coin margined futures index price
type CFuturesIndexPriceConstituent struct {
	Exchange string       `json:"exchange"`
	Symbol   string       `json:"symbol"`
	Price    types.Number `json:"price"`
	Weight   types.Number `json:"weight"`
}

// SymbolOrderBookTicker holds the best price and quantity on a coin margined futures order book
type SymbolOrderBookTicker struct {
	LastUpdateID uint64       `json:"lastUpdateId"`
	Symbol       string       `json:"symbol"`
	Pair         string       `json:"pair"`
	BidPrice     types.Number `json:"bidPrice"`
	BidQuantity  types.Number `json:"bidQty"`
	AskPrice     types.Number `json:"askPrice"`
	AskQuantity  types.Number `json:"askQty"`
	Time         types.Time   `json:"time"`
}

// SymbolPriceTicker holds the latest price of a coin margined futures symbol
type SymbolPriceTicker struct {
	Symbol string       `json:"symbol"`
	Pair   string       `json:"ps"`
	Price  types.Number `json:"price"`
	Time   types.Time   `json:"time"`
}

// TakerBuySellVolume holds coin margined futures taker buy and sell volumes
type TakerBuySellVolume struct {
	Pair         string `json:"pair"`
	ContractType string `json:"contractType"`
	// TakerBuyVolume and TakerSellVolume count contracts; the value fields are in the base asset
	TakerBuyVolume       types.Number `json:"takerBuyVol"`
	TakerSellVolume      types.Number `json:"takerSellVol"`
	TakerBuyVolumeValue  types.Number `json:"takerBuyVolValue"`
	TakerSellVolumeValue types.Number `json:"takerSellVolValue"`
	Timestamp            types.Time   `json:"timestamp"`
}

// CFuturesPriceChangeStats holds the 24 hour price change statistics of a coin margined futures symbol. Volume counts
// contracts and BaseVolume is in the base asset; coin margined futures sends no quote volume
type CFuturesPriceChangeStats struct {
	Symbol               string       `json:"symbol"`
	Pair                 string       `json:"pair"`
	PriceChange          types.Number `json:"priceChange"`
	PriceChangePercent   types.Number `json:"priceChangePercent"`
	WeightedAveragePrice types.Number `json:"weightedAvgPrice"`
	LastPrice            types.Number `json:"lastPrice"`
	LastQuantity         types.Number `json:"lastQty"`
	OpenPrice            types.Number `json:"openPrice"`
	HighPrice            types.Number `json:"highPrice"`
	LowPrice             types.Number `json:"lowPrice"`
	Volume               types.Number `json:"volume"`
	BaseVolume           types.Number `json:"baseVolume"`
	OpenTime             types.Time   `json:"openTime"`
	CloseTime            types.Time   `json:"closeTime"`
	FirstID              uint64       `json:"firstId"`
	LastID               uint64       `json:"lastId"`
	Count                uint64       `json:"count"`
}

// TopTraderPositionRatio holds a coin margined futures top trader long/short position ratio
type TopTraderPositionRatio struct {
	Pair           string       `json:"pair"`
	LongShortRatio types.Number `json:"longShortRatio"`
	LongPosition   types.Number `json:"longPosition"`
	ShortPosition  types.Number `json:"shortPosition"`
	Timestamp      types.Time   `json:"timestamp"`
}

// CFuturesAccountTradeListRequest holds the parameters for querying coin margined futures account trades
type CFuturesAccountTradeListRequest struct {
	Symbol currency.Pair
	// Pair is the contract pair, such as BTCUSD
	Pair      currency.Code
	OrderID   uint64
	StartTime time.Time
	EndTime   time.Time
	FromID    uint64
	Limit     uint64
}

// FuturesAccountTradeList holds a coin margined futures account trade
type FuturesAccountTradeList struct {
	Symbol      string        `json:"symbol"`
	ID          uint64        `json:"id"`
	OrderID     uint64        `json:"orderId"`
	Pair        string        `json:"pair"`
	Side        string        `json:"side"`
	Price       types.Number  `json:"price"`
	Quantity    types.Number  `json:"qty"`
	RealizedPNL types.Number  `json:"realizedPnl"`
	MarginAsset currency.Code `json:"marginAsset"`
	// BaseQuantity is set for coin margined symbols and QuoteQuantity for USDⓈ-M symbols, which this endpoint also
	// returns since the two futures types were integrated
	BaseQuantity    types.Number  `json:"baseQty"`
	QuoteQuantity   types.Number  `json:"quoteQty"`
	Commission      types.Number  `json:"commission"`
	CommissionAsset currency.Code `json:"commissionAsset"`
	Time            types.Time    `json:"time"`
	PositionSide    string        `json:"positionSide"`
	Buyer           bool          `json:"buyer"`
	Maker           bool          `json:"maker"`
}

// CFuturesAllOrdersRequest holds the parameters for querying all coin margined futures orders
type CFuturesAllOrdersRequest struct {
	Symbol currency.Pair
	// Pair is the contract pair, such as BTCUSD
	Pair      currency.Code
	OrderID   uint64
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
}

// FuturesOrderDetailResponse holds a coin margined futures order as the order query endpoints return it
type FuturesOrderDetailResponse struct {
	AveragePrice     types.Number `json:"avgPrice"`
	ClientOrderID    string       `json:"clientOrderId"`
	CumulativeBase   types.Number `json:"cumBase"`
	ExecutedQuantity types.Number `json:"executedQty"`
	OrderID          uint64       `json:"orderId"`
	OriginalQuantity types.Number `json:"origQty"`
	OriginalType     string       `json:"origType"`
	Price            types.Number `json:"price"`
	ReduceOnly       bool         `json:"reduceOnly"`
	Side             string       `json:"side"`
	PositionSide     string       `json:"positionSide"`
	Status           string       `json:"status"`
	StopPrice        types.Number `json:"stopPrice"`
	ClosePosition    bool         `json:"closePosition"`
	Symbol           string       `json:"symbol"`
	Pair             string       `json:"pair"`
	Time             types.Time   `json:"time"`
	TimeInForce      string       `json:"timeInForce"`
	OrderType        string       `json:"type"`
	// ActivatePrice and PriceRate are only set for TRAILING_STOP_MARKET orders
	ActivatePrice           types.Number `json:"activatePrice"`
	PriceRate               types.Number `json:"priceRate"`
	UpdateTime              types.Time   `json:"updateTime"`
	WorkingType             string       `json:"workingType"`
	PriceProtect            bool         `json:"priceProtect"`
	PriceMatch              string       `json:"priceMatch"`
	SelfTradePreventionMode string       `json:"selfTradePreventionMode"`
}

// FuturesOrderData holds a coin margined futures order as the all orders endpoint returns it
type FuturesOrderData struct {
	FuturesOrderDetailResponse
	CumulativeQuote types.Number `json:"cumQuote"`
	GoodTillDate    types.Time   `json:"goodTillDate"`
}

// CFuturesAutoCancelAllOpenOrdersResponse holds a coin margined futures auto-cancel countdown
type CFuturesAutoCancelAllOpenOrdersResponse struct {
	Symbol string `json:"symbol"`
	// CountdownTime is in milliseconds, quoted on the wire; 0 means the countdown is off
	CountdownTime types.Number `json:"countdownTime"`
}

// CFuturesStatusResponse holds the status Binance answers coin margined futures actions with
type CFuturesStatusResponse struct {
	Code    int64  `json:"code"`
	Message string `json:"msg"`
}

// FuturesOrderResponse holds a coin margined futures order as order placement and cancellation return it
type FuturesOrderResponse struct {
	ClientOrderID      string       `json:"clientOrderId"`
	CumulativeQuantity types.Number `json:"cumQty"`
	ExecutedQuantity   types.Number `json:"executedQty"`
	OrderID            uint64       `json:"orderId"`
	OriginalQuantity   types.Number `json:"origQty"`
	Price              types.Number `json:"price"`
	ReduceOnly         bool         `json:"reduceOnly"`
	ClosePosition      bool         `json:"closePosition"`
	Side               string       `json:"side"`
	PositionSide       string       `json:"positionSide"`
	Status             string       `json:"status"`
	StopPrice          types.Number `json:"stopPrice"`
	Symbol             string       `json:"symbol"`
	Pair               string       `json:"pair"`
	TimeInForce        string       `json:"timeInForce"`
	OrderType          string       `json:"type"`
	OriginalType       string       `json:"origType"`
	// ActivatePrice and PriceRate are only set for TRAILING_STOP_MARKET orders
	ActivatePrice           types.Number `json:"activatePrice"`
	PriceRate               types.Number `json:"priceRate"`
	UpdateTime              types.Time   `json:"updateTime"`
	WorkingType             string       `json:"workingType"`
	PriceProtect            bool         `json:"priceProtect"`
	PriceMatch              string       `json:"priceMatch"`
	SelfTradePreventionMode string       `json:"selfTradePreventionMode"`
}

// CFuturesModifyOrderRequest holds the parameters of Modify Order and one order of Modify Multiple Orders
type CFuturesModifyOrderRequest struct {
	Symbol            currency.Pair
	OrderID           uint64 // OrderID wins when both IDs are set
	OrigClientOrderID string
	Side              string
	Quantity          float64
	Price             float64 // required unless PriceMatch is set, and cannot be sent with it
	PriceMatch        string  // Modify Multiple Orders does not take it
	ModifyID          uint64  // returned unchanged in the reply
}

// CFuturesModifyOrderResponse holds a coin margined futures order as Modify Order returns it
type CFuturesModifyOrderResponse struct {
	OrderID                 uint64       `json:"orderId"`
	Symbol                  string       `json:"symbol"`
	Pair                    string       `json:"pair"`
	Status                  string       `json:"status"`
	ClientOrderID           string       `json:"clientOrderId"`
	ModifyID                uint64       `json:"modifyId"` // only sent when the request set it
	Price                   types.Number `json:"price"`
	OriginalQuantity        types.Number `json:"origQty"`
	ExecutedQuantity        types.Number `json:"executedQty"`
	CumulativeQuantity      types.Number `json:"cumQty"`
	TimeInForce             string       `json:"timeInForce"`
	OrderType               string       `json:"type"`
	ReduceOnly              bool         `json:"reduceOnly"`
	ClosePosition           bool         `json:"closePosition"`
	Side                    string       `json:"side"`
	PositionSide            string       `json:"positionSide"`
	StopPrice               types.Number `json:"stopPrice"`
	WorkingType             string       `json:"workingType"`
	PriceProtect            bool         `json:"priceProtect"`
	OriginalType            string       `json:"origType"`
	PriceMatch              string       `json:"priceMatch"`
	SelfTradePreventionMode string       `json:"selfTradePreventionMode"`
	UpdateTime              types.Time   `json:"updateTime"`
}

// CFuturesModifyBatchOrderResponse holds one order's reply from Modify Multiple Orders: the modified order, or the code
// and message that rejected it
type CFuturesModifyBatchOrderResponse struct {
	CFuturesModifyOrderResponse
	Code    int64  `json:"code"`
	Message string `json:"msg"`
}

// CFuturesOrderModifyHistoryRequest holds the parameters of Get Order Modify History
type CFuturesOrderModifyHistoryRequest struct {
	Symbol            currency.Pair
	OrderID           uint64 // OrderID wins when both IDs are set
	OrigClientOrderID string
	StartTime         time.Time
	EndTime           time.Time
	Limit             uint64 // 50 when zero, at most 100
}

// CFuturesBatchOrderData holds an entry of a coin margined futures batch placement or cancellation, which carries
// Code and Message instead of the order when that entry was rejected
type CFuturesBatchOrderData struct {
	FuturesOrderResponse
	Code    int64  `json:"code"`
	Message string `json:"msg"`
}

// FuturesNewOrderRequest holds the parameters for placing a coin margined futures order
type FuturesNewOrderRequest struct {
	Symbol           currency.Pair
	Side             string
	PositionSide     string
	OrderType        string
	TimeInForce      string
	Quantity         float64
	Price            float64
	ReduceOnly       bool
	NewClientOrderID string
	StopPrice        float64
	ClosePosition    bool
	ActivationPrice  float64
	// CallbackRate is a percentage, where 1 means 1%
	CallbackRate            float64
	WorkingType             string
	PriceProtect            bool
	NewOrderRespType        string
	PriceMatch              string
	SelfTradePreventionMode string
}

// CFuturesChangeLeverageResponse holds the initial leverage set for a coin margined futures symbol
type CFuturesChangeLeverageResponse struct {
	Leverage    uint64       `json:"leverage"`
	MaxQuantity types.Number `json:"maxQty"`
	Symbol      string       `json:"symbol"`
}

// CFuturesPositionMarginHistoryRequest holds the parameters for querying coin margined futures position margin changes
type CFuturesPositionMarginHistoryRequest struct {
	Symbol currency.Pair
	// Type filters by direction: 1 adds position margin and 2 reduces it
	Type      uint64
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
}

// GetPositionMarginChangeHistoryData holds a coin margined futures position margin change
type GetPositionMarginChangeHistoryData struct {
	Amount types.Number  `json:"amount"`
	Asset  currency.Code `json:"asset"`
	Symbol string        `json:"symbol"`
	Time   types.Time    `json:"time"`
	// Type is 1 when margin was added and 2 when it was reduced
	Type         uint64 `json:"type"`
	PositionSide string `json:"positionSide"`
}

// CFuturesModifyIsolatedPositionMarginRequest holds the parameters for changing a coin margined futures isolated
// position's margin
type CFuturesModifyIsolatedPositionMarginRequest struct {
	Symbol       currency.Pair
	PositionSide string
	Amount       float64
	// Type is 1 to add position margin and 2 to reduce it
	Type uint64
}

// FuturesMarginUpdatedResponse holds the result of changing a coin margined futures isolated position's margin
type FuturesMarginUpdatedResponse struct {
	Amount  float64 `json:"amount"`
	Code    int64   `json:"code"`
	Message string  `json:"msg"`
	// Type is 1 when margin was added and 2 when it was reduced
	Type uint64 `json:"type"`
}

// ADLEstimateData holds the auto-deleveraging queue estimate of a coin margined futures symbol's positions
type ADLEstimateData struct {
	Symbol      string              `json:"symbol"`
	ADLQuantile CFuturesADLQuantile `json:"adlQuantile"`
}

// CFuturesADLQuantile holds auto-deleveraging quantiles from 0 (lowest) to 4 (highest) for each position side.
// Crossed hedge mode positions report HEDGE instead of BOTH
type CFuturesADLQuantile struct {
	Long  uint64 `json:"LONG"`
	Short uint64 `json:"SHORT"`
	Hedge uint64 `json:"HEDGE"`
	Both  uint64 `json:"BOTH"`
}

// FuturesPositionInformation holds a coin margined futures position
type FuturesPositionInformation struct {
	Symbol           string       `json:"symbol"`
	PositionAmount   types.Number `json:"positionAmt"`
	EntryPrice       types.Number `json:"entryPrice"`
	BreakEvenPrice   types.Number `json:"breakEvenPrice"`
	MarkPrice        types.Number `json:"markPrice"`
	UnrealizedProfit types.Number `json:"unRealizedProfit"`
	LiquidationPrice types.Number `json:"liquidationPrice"`
	Leverage         types.Number `json:"leverage"`
	MaxQuantity      types.Number `json:"maxQty"`
	MarginType       string       `json:"marginType"`
	IsolatedMargin   types.Number `json:"isolatedMargin"`
	// IsAutoAddMargin is quoted on the wire
	IsAutoAddMargin types.Boolean `json:"isAutoAddMargin"`
	PositionSide    string        `json:"positionSide"`
	UpdateTime      types.Time    `json:"updateTime"`
	// NotionalValue and IsolatedWallet are missing from the REST docs, but the WebSocket API's account.position schema
	// for the same position data documents them
	NotionalValue  types.Number `json:"notionalValue"`
	IsolatedWallet types.Number `json:"isolatedWallet"`
}

// CFuturesForceOrdersRequest holds the parameters for querying the coin margined futures orders Binance forced
type CFuturesForceOrdersRequest struct {
	Symbol        currency.Pair
	AutoCloseType string
	StartTime     time.Time
	EndTime       time.Time
	Limit         uint64
}

// ForcedOrdersData holds a coin margined futures order Binance placed to liquidate or deleverage a position
type ForcedOrdersData struct {
	OrderID          uint64       `json:"orderId"`
	Symbol           string       `json:"symbol"`
	Pair             string       `json:"pair"`
	Status           string       `json:"status"`
	ClientOrderID    string       `json:"clientOrderId"`
	Price            types.Number `json:"price"`
	AveragePrice     types.Number `json:"avgPrice"`
	OriginalQuantity types.Number `json:"origQty"`
	ExecutedQuantity types.Number `json:"executedQty"`
	CumulativeBase   types.Number `json:"cumBase"`
	CumulativeQuote  types.Number `json:"cumQuote"`
	TimeInForce      string       `json:"timeInForce"`
	OrderType        string       `json:"type"`
	ReduceOnly       bool         `json:"reduceOnly"`
	ClosePosition    bool         `json:"closePosition"`
	Side             string       `json:"side"`
	PositionSide     string       `json:"positionSide"`
	StopPrice        types.Number `json:"stopPrice"`
	WorkingType      string       `json:"workingType"`
	PriceProtect     bool         `json:"priceProtect"`
	OriginalType     string       `json:"origType"`
	Time             types.Time   `json:"time"`
	UpdateTime       types.Time   `json:"updateTime"`
	GoodTillDate     types.Time   `json:"goodTillDate"`
}
