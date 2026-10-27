package binance

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// CoinMarginedFuturesAccountInformationResponse holds the coin margined futures account information
type CoinMarginedFuturesAccountInformationResponse struct {
	Assets      []CoinMarginedFuturesAccountAsset               `json:"assets"`
	Positions   []CoinMarginedFuturesAccountInformationPosition `json:"positions"`
	CanDeposit  bool                                            `json:"canDeposit"`
	CanTrade    bool                                            `json:"canTrade"`
	CanWithdraw bool                                            `json:"canWithdraw"`
	FeeTier     uint64                                          `json:"feeTier"`
	UpdateTime  types.Time                                      `json:"updateTime"`
}

// CoinMarginedFuturesAccountAsset holds a coin margined futures account asset balance
type CoinMarginedFuturesAccountAsset struct {
	Asset                  currency.Code `json:"asset"`
	WalletBalance          types.Number  `json:"walletBalance"`
	UnrealisedProfit       types.Number  `json:"unrealizedProfit"`
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

// CoinMarginedFuturesAccountInformationPosition holds a coin margined futures account position
type CoinMarginedFuturesAccountInformationPosition struct {
	Symbol                 string       `json:"symbol"`
	PositionAmount         types.Number `json:"positionAmt"`
	InitialMargin          types.Number `json:"initialMargin"`
	MaintenanceMargin      types.Number `json:"maintMargin"`
	UnrealisedProfit       types.Number `json:"unrealizedProfit"`
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

// CoinMarginedFuturesAccountBalanceResponse holds a coin margined futures account balance
type CoinMarginedFuturesAccountBalanceResponse struct {
	AccountAlias       string        `json:"accountAlias"`
	Asset              currency.Code `json:"asset"`
	Balance            types.Number  `json:"balance"`
	WithdrawAvailable  types.Number  `json:"withdrawAvailable"`
	CrossWalletBalance types.Number  `json:"crossWalletBalance"`
	CrossUnPNL         types.Number  `json:"crossUnPnl"`
	AvailableBalance   types.Number  `json:"availableBalance"`
	UpdateTime         types.Time    `json:"updateTime"`
}

// CoinMarginedFuturesIncomeHistoryRequest holds the parameters for querying coin margined futures income history
type CoinMarginedFuturesIncomeHistoryRequest struct {
	Symbol     currency.Pair
	IncomeType string
	StartTime  time.Time
	EndTime    time.Time
	Page       uint64
	Limit      uint64
}

// CoinMarginedFuturesIncomeHistoryResponse holds a coin margined futures income record
type CoinMarginedFuturesIncomeHistoryResponse struct {
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

// NotionalBracketResponse holds the notional brackets of a coin margined futures pair
type NotionalBracketResponse struct {
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

// CoinMarginedFuturesStatisticsRequest holds the parameters shared by the coin margined futures statistics endpoints
// under /futures/data
type CoinMarginedFuturesStatisticsRequest struct {
	// Pair is the contract pair, such as BTCUSD
	Pair         currency.Code
	ContractType string
	Period       kline.Interval
	Limit        uint64
	StartTime    time.Time
	EndTime      time.Time
}

// CoinMarginedFuturesBasisResponse holds a coin margined futures basis record
type CoinMarginedFuturesBasisResponse struct {
	IndexPrice   types.Number `json:"indexPrice"`
	ContractType string       `json:"contractType"`
	BasisRate    types.Number `json:"basisRate"`
	FuturesPrice types.Number `json:"futuresPrice"`
	// AnnualisedBasisRate is empty for perpetual contracts
	AnnualisedBasisRate types.Number `json:"annualizedBasisRate"`
	Basis               types.Number `json:"basis"`
	Pair                string       `json:"pair"`
	Timestamp           types.Time   `json:"timestamp"`
}

// CoinMarginedFuturesAggregatedTradesRequest holds the parameters for querying coin margined futures aggregate trades
type CoinMarginedFuturesAggregatedTradesRequest struct {
	Symbol    currency.Pair
	FromID    uint64
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
}

// CoinMarginedFuturesAggregatedTradeResponse holds a coin margined futures aggregate trade
type CoinMarginedFuturesAggregatedTradeResponse struct {
	AggregateTradeID uint64       `json:"a"`
	Price            types.Number `json:"p"`
	Quantity         types.Number `json:"q"`
	FirstTradeID     uint64       `json:"f"`
	LastTradeID      uint64       `json:"l"`
	Timestamp        types.Time   `json:"T"`
	IsBuyerMaker     bool         `json:"m"`
}

// CoinMarginedFuturesContinuousKlineRequest holds the parameters for querying coin margined futures continuous contract
// klines
type CoinMarginedFuturesContinuousKlineRequest struct {
	// Pair is the contract pair, such as BTCUSD
	Pair         currency.Code
	ContractType string
	Interval     kline.Interval
	StartTime    time.Time
	EndTime      time.Time
	Limit        uint64
}

// CoinMarginedFuturesCandleStickResponse holds a coin margined futures kline
type CoinMarginedFuturesCandleStickResponse struct {
	OpenTime                  types.Time
	Open                      types.Number
	High                      types.Number
	Low                       types.Number
	Close                     types.Number
	VolumeInContracts         types.Number
	CloseTime                 types.Time
	BaseAssetVolume           types.Number
	NumberOfTrades            uint64
	TakerBuyVolumeInContracts types.Number
	TakerBuyBaseAssetVolume   types.Number
}

// UnmarshalJSON decodes a coin margined futures kline from its array form
func (c *CoinMarginedFuturesCandleStickResponse) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[12]any{&c.OpenTime, &c.Open, &c.High, &c.Low, &c.Close, &c.VolumeInContracts, &c.CloseTime, &c.BaseAssetVolume, &c.NumberOfTrades, &c.TakerBuyVolumeInContracts, &c.TakerBuyBaseAssetVolume, nil})
}

// CoinMarginedFuturesExchangeInfoResponse holds the coin margined futures trading rules and symbol information
type CoinMarginedFuturesExchangeInfoResponse struct {
	// ExchangeFilters is always empty in practice
	ExchangeFilters []string                        `json:"exchangeFilters"`
	RateLimits      []RateLimitInfoResponse         `json:"rateLimits"`
	ServerTime      types.Time                      `json:"serverTime"`
	Symbols         []CoinMarginedFuturesSymbolInfo `json:"symbols"`
	Timezone        string                          `json:"timezone"`
}

// CoinMarginedFuturesSymbolInfo holds the trading rules of a coin margined futures symbol
type CoinMarginedFuturesSymbolInfo struct {
	Filters        []CoinMarginedFuturesSymbolFilter `json:"filters"`
	OrderTypes     []string                          `json:"orderTypes"`
	TimeInForce    []string                          `json:"timeInForce"`
	LiquidationFee types.Number                      `json:"liquidationFee"`
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

// CoinMarginedFuturesSymbolFilter holds a coin margined futures symbol filter; the fields set depend on FilterType
type CoinMarginedFuturesSymbolFilter struct {
	FilterType        string       `json:"filterType"`
	MaxPrice          types.Number `json:"maxPrice"`
	MinPrice          types.Number `json:"minPrice"`
	TickSize          types.Number `json:"tickSize"`
	MaxQuantity       types.Number `json:"maxQty"`
	MinQuantity       types.Number `json:"minQty"`
	StepSize          types.Number `json:"stepSize"`
	Limit             uint64       `json:"limit"`
	MultiplierUp      types.Number `json:"multiplierUp"`
	MultiplierDown    types.Number `json:"multiplierDown"`
	MultiplierDecimal types.Number `json:"multiplierDecimal"`
}

// CoinMarginedFuturesFundingRateHistoryRequest holds the parameters for querying coin margined perpetual futures
// funding rates
type CoinMarginedFuturesFundingRateHistoryRequest struct {
	Symbol    currency.Pair
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
}

// CoinMarginedFuturesFundingRateHistoryResponse holds a coin margined perpetual futures funding rate
type CoinMarginedFuturesFundingRateHistoryResponse struct {
	Symbol      string       `json:"symbol"`
	FundingTime types.Time   `json:"fundingTime"`
	FundingRate types.Number `json:"fundingRate"`
	// MarkPrice and RateType are sent since the coin margined and USDⓈ-M futures integration but not documented
	MarkPrice types.Number `json:"markPrice"`
	RateType  string       `json:"rateType"`
}

// CoinMarginedFuturesFundingRateInfoResponse holds the funding rate limits of a coin margined perpetual futures symbol
// whose cap, floor or funding interval was adjusted
type CoinMarginedFuturesFundingRateInfoResponse struct {
	Symbol                   string       `json:"symbol"`
	AdjustedFundingRateCap   types.Number `json:"adjustedFundingRateCap"`
	AdjustedFundingRateFloor types.Number `json:"adjustedFundingRateFloor"`
	FundingIntervalHours     uint64       `json:"fundingIntervalHours"`
	Disclaimer               bool         `json:"disclaimer"`
}

// IndexMarkPriceResponse holds the index and mark prices of a coin margined futures symbol
type IndexMarkPriceResponse struct {
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

// CoinMarginedFuturesIndexPriceKlineRequest holds the parameters for querying coin margined futures index price klines
type CoinMarginedFuturesIndexPriceKlineRequest struct {
	// Pair is the contract pair, such as BTCUSD
	Pair      currency.Code
	Interval  kline.Interval
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
}

// CoinMarginedFuturesPriceCandleStickResponse holds a coin margined futures index or mark price kline
type CoinMarginedFuturesPriceCandleStickResponse struct {
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
func (c *CoinMarginedFuturesPriceCandleStickResponse) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[12]any{&c.OpenTime, &c.Open, &c.High, &c.Low, &c.Close, nil, &c.CloseTime, nil, &c.NumberOfBasicData, nil, nil, nil})
}

// CoinMarginedFuturesKlineRequest holds the parameters for querying coin margined futures klines, mark price klines and
// premium index klines of a symbol
type CoinMarginedFuturesKlineRequest struct {
	Symbol    currency.Pair
	Interval  kline.Interval
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
}

// CoinMarginedFuturesLongShortAccountRatioResponse holds a coin margined futures long/short account ratio, of the top
// traders or of every account
type CoinMarginedFuturesLongShortAccountRatioResponse struct {
	Pair           string       `json:"pair"`
	LongShortRatio types.Number `json:"longShortRatio"`
	LongAccount    types.Number `json:"longAccount"`
	ShortAccount   types.Number `json:"shortAccount"`
	Timestamp      types.Time   `json:"timestamp"`
}

// CoinMarginedFuturesPublicTradesResponse holds a coin margined futures trade; coin margined futures sends no quote
// figure
type CoinMarginedFuturesPublicTradesResponse struct {
	ID                  uint64       `json:"id"`
	Price               types.Number `json:"price"`
	QuantityInContracts types.Number `json:"qty"`
	BaseQuantity        types.Number `json:"baseQty"`
	Time                types.Time   `json:"time"`
	IsBuyerMaker        bool         `json:"isBuyerMaker"`
}

// CoinMarginedFuturesOpenInterestResponse holds the open interest of a coin margined futures symbol
type CoinMarginedFuturesOpenInterestResponse struct {
	Symbol       string       `json:"symbol"`
	Pair         string       `json:"pair"`
	OpenInterest types.Number `json:"openInterest"`
	ContractType string       `json:"contractType"`
	Time         types.Time   `json:"time"`
}

// OpenInterestStatsResponse holds a coin margined futures open interest statistic
type OpenInterestStatsResponse struct {
	Pair                            string       `json:"pair"`
	ContractType                    string       `json:"contractType"`
	SumOpenInterestInContracts      types.Number `json:"sumOpenInterest"`
	SumOpenInterestValueInBaseAsset types.Number `json:"sumOpenInterestValue"`
	Timestamp                       types.Time   `json:"timestamp"`
}

// CoinMarginedFuturesOrderBookResponse holds a coin margined futures order book
type CoinMarginedFuturesOrderBookResponse struct {
	LastUpdateID      uint64                           `json:"lastUpdateId"`
	Symbol            string                           `json:"symbol"`
	Pair              string                           `json:"pair"`
	MessageOutputTime types.Time                       `json:"E"`
	TransactionTime   types.Time                       `json:"T"`
	Bids              orderbook.LevelsArrayPriceAmount `json:"bids"`
	Asks              orderbook.LevelsArrayPriceAmount `json:"asks"`
}

// CoinMarginedFuturesPremiumIndexCandleStickResponse holds a coin margined futures premium index kline
type CoinMarginedFuturesPremiumIndexCandleStickResponse struct {
	OpenTime  types.Time
	Open      types.Number
	High      types.Number
	Low       types.Number
	Close     types.Number
	CloseTime types.Time
}

// UnmarshalJSON decodes a coin margined futures premium index kline from its array form
func (c *CoinMarginedFuturesPremiumIndexCandleStickResponse) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[12]any{&c.OpenTime, &c.Open, &c.High, &c.Low, &c.Close, nil, &c.CloseTime, nil, nil, nil, nil, nil})
}

// CoinMarginedFuturesIndexPriceConstituentsResponse holds the constituents of a coin margined futures index price
type CoinMarginedFuturesIndexPriceConstituentsResponse struct {
	Symbol       string                                     `json:"symbol"`
	Time         types.Time                                 `json:"time"`
	Constituents []CoinMarginedFuturesIndexPriceConstituent `json:"constituents"`
}

// CoinMarginedFuturesIndexPriceConstituent holds an exchange price that makes up a coin margined futures index price
type CoinMarginedFuturesIndexPriceConstituent struct {
	Exchange string       `json:"exchange"`
	Symbol   string       `json:"symbol"`
	Price    types.Number `json:"price"`
	Weight   types.Number `json:"weight"`
}

// SymbolOrderBookTickerResponse holds the best price and quantity on a coin margined futures order book
type SymbolOrderBookTickerResponse struct {
	LastUpdateID uint64       `json:"lastUpdateId"`
	Symbol       string       `json:"symbol"`
	Pair         string       `json:"pair"`
	BidPrice     types.Number `json:"bidPrice"`
	BidQuantity  types.Number `json:"bidQty"`
	AskPrice     types.Number `json:"askPrice"`
	AskQuantity  types.Number `json:"askQty"`
	Time         types.Time   `json:"time"`
}

// SymbolPriceTickerResponse holds the latest price of a coin margined futures symbol
type SymbolPriceTickerResponse struct {
	Symbol string       `json:"symbol"`
	Pair   string       `json:"ps"`
	Price  types.Number `json:"price"`
	Time   types.Time   `json:"time"`
}

// TakerBuySellVolumeResponse holds coin margined futures taker buy and sell volumes
type TakerBuySellVolumeResponse struct {
	Pair                            string       `json:"pair"`
	ContractType                    string       `json:"contractType"`
	TakerBuyVolumeInContracts       types.Number `json:"takerBuyVol"`
	TakerSellVolumeInContracts      types.Number `json:"takerSellVol"`
	TakerBuyVolumeValueInBaseAsset  types.Number `json:"takerBuyVolValue"`
	TakerSellVolumeValueInBaseAsset types.Number `json:"takerSellVolValue"`
	Timestamp                       types.Time   `json:"timestamp"`
}

// CoinMarginedFuturesPriceChangeStatsResponse holds the 24 hour price change statistics of a coin margined futures
// symbol; coin margined futures sends no quote volume
type CoinMarginedFuturesPriceChangeStatsResponse struct {
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
	VolumeInContracts    types.Number `json:"volume"`
	BaseVolume           types.Number `json:"baseVolume"`
	OpenTime             types.Time   `json:"openTime"`
	CloseTime            types.Time   `json:"closeTime"`
	FirstID              uint64       `json:"firstId"`
	LastID               uint64       `json:"lastId"`
	Count                uint64       `json:"count"`
}

// TopTraderPositionRatioResponse holds a coin margined futures top trader long/short position ratio
type TopTraderPositionRatioResponse struct {
	Pair           string       `json:"pair"`
	LongShortRatio types.Number `json:"longShortRatio"`
	LongPosition   types.Number `json:"longPosition"`
	ShortPosition  types.Number `json:"shortPosition"`
	Timestamp      types.Time   `json:"timestamp"`
}

// CoinMarginedFuturesAccountTradeListRequest holds the parameters for querying coin margined futures account trades
type CoinMarginedFuturesAccountTradeListRequest struct {
	Symbol currency.Pair
	// Pair is the contract pair, such as BTCUSD
	Pair      currency.Code
	OrderID   uint64
	StartTime time.Time
	EndTime   time.Time
	FromID    uint64
	Limit     uint64
}

// CoinMarginedFuturesAccountTradeListResponse holds a coin margined futures account trade
type CoinMarginedFuturesAccountTradeListResponse struct {
	Symbol      string        `json:"symbol"`
	ID          uint64        `json:"id"`
	OrderID     uint64        `json:"orderId"`
	Pair        string        `json:"pair"`
	Side        string        `json:"side"`
	Price       types.Number  `json:"price"`
	Quantity    types.Number  `json:"qty"`
	RealisedPNL types.Number  `json:"realizedPnl"`
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

// CoinMarginedFuturesAllOrdersRequest holds the parameters for querying all coin margined futures orders
type CoinMarginedFuturesAllOrdersRequest struct {
	Symbol currency.Pair
	// Pair is the contract pair, such as BTCUSD
	Pair      currency.Code
	OrderID   uint64
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
}

// CoinMarginedFuturesOrderDetailResponse holds a coin margined futures order as the order query endpoints return it
type CoinMarginedFuturesOrderDetailResponse struct {
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

// CoinMarginedFuturesAllOrdersResponse holds a coin margined futures order as the all orders endpoint returns it
type CoinMarginedFuturesAllOrdersResponse struct {
	CoinMarginedFuturesOrderDetailResponse
	CumulativeQuote types.Number `json:"cumQuote"`
	GoodTillDate    types.Time   `json:"goodTillDate"`
}

// CoinMarginedFuturesAutoCancelAllOpenOrdersResponse holds a coin margined futures auto-cancel countdown
type CoinMarginedFuturesAutoCancelAllOpenOrdersResponse struct {
	Symbol string `json:"symbol"`
	// CountdownTimeInMilliseconds is 0 when the countdown is off
	CountdownTimeInMilliseconds types.Number `json:"countdownTime"`
}

// CoinMarginedFuturesStatusResponse holds the status Binance answers coin margined futures actions with
type CoinMarginedFuturesStatusResponse struct {
	Code    int64  `json:"code"` // 200 on success; Binance's error codes are negative
	Message string `json:"msg"`
}

// CoinMarginedFuturesOrderResponse holds a coin margined futures order as order placement and cancellation return it
type CoinMarginedFuturesOrderResponse struct {
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

// CoinMarginedFuturesModifyOrderRequest holds the parameters of Modify Order and one order of Modify Multiple Orders
type CoinMarginedFuturesModifyOrderRequest struct {
	Symbol            currency.Pair
	OrderID           uint64 // OrderID wins when both IDs are set
	OrigClientOrderID string
	Side              string
	Quantity          float64
	Price             float64 // required unless PriceMatch is set, and cannot be sent with it
	PriceMatch        string  // Modify Multiple Orders does not take it
	ModifyID          uint64  // returned unchanged in the reply
}

// CoinMarginedFuturesModifyOrderResponse holds a coin margined futures order as Modify Order returns it
type CoinMarginedFuturesModifyOrderResponse struct {
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

// CoinMarginedFuturesModifyBatchOrderResponse holds one order's reply from Modify Multiple Orders: the modified order,
// or the code and message that rejected it
type CoinMarginedFuturesModifyBatchOrderResponse struct {
	CoinMarginedFuturesModifyOrderResponse
	Code    int64  `json:"code"` // negative when Binance rejected the entry
	Message string `json:"msg"`
}

// CoinMarginedFuturesOrderModifyHistoryRequest holds the parameters of Get Order Modify History
type CoinMarginedFuturesOrderModifyHistoryRequest struct {
	Symbol            currency.Pair
	OrderID           uint64 // OrderID wins when both IDs are set
	OrigClientOrderID string
	StartTime         time.Time
	EndTime           time.Time
	Limit             uint64 // 50 when zero, at most 100
}

// CoinMarginedFuturesBatchOrderResponse holds an entry of a coin margined futures batch placement or cancellation,
// which carries Code and Message instead of the order when that entry was rejected
type CoinMarginedFuturesBatchOrderResponse struct {
	CoinMarginedFuturesOrderResponse
	Code    int64  `json:"code"` // negative when Binance rejected the entry
	Message string `json:"msg"`
}

// CoinMarginedFuturesNewOrderRequest holds the parameters for placing a coin margined futures order
type CoinMarginedFuturesNewOrderRequest struct {
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
	NewOrderResponseType    string
	PriceMatch              string
	SelfTradePreventionMode string
}

// CoinMarginedFuturesChangeLeverageResponse holds the initial leverage set for a coin margined futures symbol
type CoinMarginedFuturesChangeLeverageResponse struct {
	Leverage    uint64       `json:"leverage"`
	MaxQuantity types.Number `json:"maxQty"`
	Symbol      string       `json:"symbol"`
}

// CoinMarginedFuturesPositionMarginHistoryRequest holds the parameters for querying coin margined futures position
// margin changes
type CoinMarginedFuturesPositionMarginHistoryRequest struct {
	Symbol currency.Pair
	// Type filters by direction: 1 adds position margin and 2 reduces it
	Type      uint64
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
}

// GetPositionMarginChangeHistoryResponse holds a coin margined futures position margin change
type GetPositionMarginChangeHistoryResponse struct {
	Amount types.Number  `json:"amount"`
	Asset  currency.Code `json:"asset"`
	Symbol string        `json:"symbol"`
	Time   types.Time    `json:"time"`
	// Type is 1 when margin was added and 2 when it was reduced
	Type         uint64 `json:"type"`
	PositionSide string `json:"positionSide"`
}

// CoinMarginedFuturesModifyIsolatedPositionMarginRequest holds the parameters for changing a coin margined futures
// isolated position's margin
type CoinMarginedFuturesModifyIsolatedPositionMarginRequest struct {
	Symbol       currency.Pair
	PositionSide string
	Amount       float64
	// Type is 1 to add position margin and 2 to reduce it
	Type uint64
}

// CoinMarginedFuturesMarginUpdatedResponse holds the result of changing a coin margined futures isolated position's
// margin
type CoinMarginedFuturesMarginUpdatedResponse struct {
	Amount  float64 `json:"amount"`
	Code    int64   `json:"code"` // 200 on success; Binance's error codes are negative
	Message string  `json:"msg"`
	// Type is 1 when margin was added and 2 when it was reduced
	Type uint64 `json:"type"`
}

// AutoDeleveragingEstimateResponse holds the auto-deleveraging queue estimate of a coin margined futures symbol's
// positions
type AutoDeleveragingEstimateResponse struct {
	Symbol                   string                                      `json:"symbol"`
	AutoDeleveragingQuantile CoinMarginedFuturesAutoDeleveragingQuantile `json:"adlQuantile"`
}

// CoinMarginedFuturesAutoDeleveragingQuantile holds auto-deleveraging quantiles from 0 (lowest) to 4 (highest) for each
// position side. Crossed hedge mode positions report HEDGE instead of BOTH
type CoinMarginedFuturesAutoDeleveragingQuantile struct {
	Long  uint64 `json:"LONG"`
	Short uint64 `json:"SHORT"`
	Hedge uint64 `json:"HEDGE"`
	Both  uint64 `json:"BOTH"`
}

// CoinMarginedFuturesPositionInformationResponse holds a coin margined futures position
type CoinMarginedFuturesPositionInformationResponse struct {
	Symbol           string       `json:"symbol"`
	PositionAmount   types.Number `json:"positionAmt"`
	EntryPrice       types.Number `json:"entryPrice"`
	BreakEvenPrice   types.Number `json:"breakEvenPrice"`
	MarkPrice        types.Number `json:"markPrice"`
	UnrealisedProfit types.Number `json:"unRealizedProfit"`
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

// CoinMarginedFuturesForceOrdersRequest holds the parameters for querying the coin margined futures orders Binance
// forced
type CoinMarginedFuturesForceOrdersRequest struct {
	Symbol        currency.Pair
	AutoCloseType string
	StartTime     time.Time
	EndTime       time.Time
	Limit         uint64
}

// ForcedOrdersResponse holds a coin margined futures order Binance placed to liquidate or deleverage a position
type ForcedOrdersResponse struct {
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
