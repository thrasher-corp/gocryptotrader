package binance

import (
	"errors"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// validMarginChange maps add and reduce to Binance's margin change types; other enums are left to the server to
// validate, so a stale list cannot reject a valid value
var (
	validMarginChange = map[string]int64{
		"add":    1,
		"reduce": 2,
	}

	errContractTypeIsRequired  = errors.New("contract type is required")
	errInvalidPeriodOrInterval = errors.New("invalid period")
)

// UAccountInformationV2Response holds the reply of Account Information V2
type UAccountInformationV2Response struct {
	FeeTier                     uint64       `json:"feeTier"`
	FeeBurn                     bool         `json:"feeBurn"`
	CanTrade                    bool         `json:"canTrade"`
	CanDeposit                  bool         `json:"canDeposit"`
	CanWithdraw                 bool         `json:"canWithdraw"`
	UpdateTime                  types.Time   `json:"updateTime"`
	MultiAssetsMargin           bool         `json:"multiAssetsMargin"`
	TradeGroupID                int64        `json:"tradeGroupId"` // -1 when the account is in no trade group
	TotalInitialMargin          types.Number `json:"totalInitialMargin"`
	TotalMaintenanceMargin      types.Number `json:"totalMaintMargin"`
	TotalWalletBalance          types.Number `json:"totalWalletBalance"`
	TotalUnrealizedProfit       types.Number `json:"totalUnrealizedProfit"`
	TotalMarginBalance          types.Number `json:"totalMarginBalance"`
	TotalPositionInitialMargin  types.Number `json:"totalPositionInitialMargin"`
	TotalOpenOrderInitialMargin types.Number `json:"totalOpenOrderInitialMargin"`
	TotalCrossWalletBalance     types.Number `json:"totalCrossWalletBalance"`
	TotalCrossUnrealizedPNL     types.Number `json:"totalCrossUnPnl"`
	AvailableBalance            types.Number `json:"availableBalance"`
	MaxWithdrawAmount           types.Number `json:"maxWithdrawAmount"`
	Assets                      []UAsset     `json:"assets"`
	Positions                   []UPosition  `json:"positions"`
}

// UAsset holds an asset's balances in Account Information V2
type UAsset struct {
	Asset                  currency.Code `json:"asset"`
	WalletBalance          types.Number  `json:"walletBalance"`
	UnrealizedProfit       types.Number  `json:"unrealizedProfit"`
	MarginBalance          types.Number  `json:"marginBalance"`
	MaintenanceMargin      types.Number  `json:"maintMargin"`
	InitialMargin          types.Number  `json:"initialMargin"`
	PositionInitialMargin  types.Number  `json:"positionInitialMargin"`
	OpenOrderInitialMargin types.Number  `json:"openOrderInitialMargin"`
	CrossWalletBalance     types.Number  `json:"crossWalletBalance"`
	CrossUnrealizedPNL     types.Number  `json:"crossUnPnl"`
	AvailableBalance       types.Number  `json:"availableBalance"`
	MaxWithdrawAmount      types.Number  `json:"maxWithdrawAmount"`
	MarginAvailable        bool          `json:"marginAvailable"`
	UpdateTime             types.Time    `json:"updateTime"`
}

// UPosition holds a symbol's position in Account Information V2
type UPosition struct {
	Symbol                 string       `json:"symbol"`
	InitialMargin          types.Number `json:"initialMargin"`
	MaintenanceMargin      types.Number `json:"maintMargin"`
	UnrealizedProfit       types.Number `json:"unrealizedProfit"`
	PositionInitialMargin  types.Number `json:"positionInitialMargin"`
	OpenOrderInitialMargin types.Number `json:"openOrderInitialMargin"`
	Leverage               types.Number `json:"leverage"`
	Isolated               bool         `json:"isolated"`
	EntryPrice             types.Number `json:"entryPrice"`
	MaxNotional            types.Number `json:"maxNotional"`
	BidNotional            types.Number `json:"bidNotional"`
	AskNotional            types.Number `json:"askNotional"`
	PositionSide           string       `json:"positionSide"`
	PositionAmount         types.Number `json:"positionAmt"`
	UpdateTime             types.Time   `json:"updateTime"`
}

// UAccountInformationV3Response holds the reply of Account Information V3, which leaves the account and symbol
// configuration to Futures Account Configuration and Symbol Configuration
type UAccountInformationV3Response struct {
	TotalInitialMargin          types.Number  `json:"totalInitialMargin"`
	TotalMaintenanceMargin      types.Number  `json:"totalMaintMargin"`
	TotalWalletBalance          types.Number  `json:"totalWalletBalance"`
	TotalUnrealizedProfit       types.Number  `json:"totalUnrealizedProfit"`
	TotalMarginBalance          types.Number  `json:"totalMarginBalance"`
	TotalPositionInitialMargin  types.Number  `json:"totalPositionInitialMargin"`
	TotalOpenOrderInitialMargin types.Number  `json:"totalOpenOrderInitialMargin"`
	TotalCrossWalletBalance     types.Number  `json:"totalCrossWalletBalance"`
	TotalCrossUnrealizedPNL     types.Number  `json:"totalCrossUnPnl"`
	AvailableBalance            types.Number  `json:"availableBalance"`
	MaxWithdrawAmount           types.Number  `json:"maxWithdrawAmount"`
	Assets                      []UAssetV3    `json:"assets"`
	Positions                   []UPositionV3 `json:"positions"`
}

// UAssetV3 holds an asset's balances in Account Information V3
type UAssetV3 struct {
	Asset                  currency.Code `json:"asset"`
	WalletBalance          types.Number  `json:"walletBalance"`
	UnrealizedProfit       types.Number  `json:"unrealizedProfit"`
	MarginBalance          types.Number  `json:"marginBalance"`
	MaintenanceMargin      types.Number  `json:"maintMargin"`
	InitialMargin          types.Number  `json:"initialMargin"`
	PositionInitialMargin  types.Number  `json:"positionInitialMargin"`
	OpenOrderInitialMargin types.Number  `json:"openOrderInitialMargin"`
	CrossWalletBalance     types.Number  `json:"crossWalletBalance"`
	CrossUnrealizedPNL     types.Number  `json:"crossUnPnl"`
	AvailableBalance       types.Number  `json:"availableBalance"`
	MaxWithdrawAmount      types.Number  `json:"maxWithdrawAmount"`
	UpdateTime             types.Time    `json:"updateTime"`
}

// UPositionV3 holds a position in Account Information V3, which only lists symbols with a position or open orders
type UPositionV3 struct {
	Symbol            string       `json:"symbol"`
	PositionSide      string       `json:"positionSide"`
	PositionAmount    types.Number `json:"positionAmt"`
	UnrealizedProfit  types.Number `json:"unrealizedProfit"`
	IsolatedMargin    types.Number `json:"isolatedMargin"`
	Notional          types.Number `json:"notional"`
	IsolatedWallet    types.Number `json:"isolatedWallet"`
	InitialMargin     types.Number `json:"initialMargin"`
	MaintenanceMargin types.Number `json:"maintMargin"`
	UpdateTime        types.Time   `json:"updateTime"`
}

// UAccountBalance holds an asset's balance from Futures Account Balance V3
type UAccountBalance struct {
	AccountAlias       string        `json:"accountAlias"`
	Asset              currency.Code `json:"asset"`
	Balance            types.Number  `json:"balance"`
	CrossWalletBalance types.Number  `json:"crossWalletBalance"`
	CrossUnrealizedPNL types.Number  `json:"crossUnPnl"`
	AvailableBalance   types.Number  `json:"availableBalance"`
	MaxWithdrawAmount  types.Number  `json:"maxWithdrawAmount"`
	MarginAvailable    bool          `json:"marginAvailable"`
	UpdateTime         types.Time    `json:"updateTime"`
}

// TradingQuantitativeRulesIndicatorsResponse holds the Futures Trading Quantitative Rules indicators, keyed by symbol, or by
// ACCOUNT for the indicators that apply to the whole account
type TradingQuantitativeRulesIndicatorsResponse struct {
	Indicators map[string][]TradingQuantitativeRulesIndicator `json:"indicators"`
	UpdateTime types.Time                                     `json:"updateTime"`
}

// TradingQuantitativeRulesIndicator holds one quantitative rule's state
type TradingQuantitativeRulesIndicator struct {
	IsLocked           bool       `json:"isLocked"`
	PlannedRecoverTime types.Time `json:"plannedRecoverTime"`
	Indicator          string     `json:"indicator"`
	Value              float64    `json:"value"`
	TriggerValue       float64    `json:"triggerValue"`
}

// UMultiAssetsModeResponse holds the reply of Get Current Multi-Assets Mode
type UMultiAssetsModeResponse struct {
	MultiAssetsMargin bool `json:"multiAssetsMargin"` // true for Multi-Assets Mode, false for Single-Asset Mode
}

// UPositionModeResponse holds the reply of Get Current Position Mode
type UPositionModeResponse struct {
	DualSidePosition bool `json:"dualSidePosition"` // true for Hedge Mode, false for One-way Mode
}

// UDownloadIDResponse holds the reply of the Get Download Id endpoints for order, trade and transaction history
type UDownloadIDResponse struct {
	AverageCostTimestampOfLast30Days uint64 `json:"avgCostTimestampOfLast30d"` // average milliseconds a download took over the last 30 days
	DownloadID                       string `json:"downloadId"`
}

// UDownloadLinkResponse holds the reply of the Download Link by Id endpoints for order, trade and transaction history
type UDownloadLinkResponse struct {
	DownloadID string `json:"downloadId"`
	Status     string `json:"status"` // completed or processing
	URL        string `json:"url"`
	Notified   bool   `json:"notified"`
	// ExpirationTimestamp is when the link expires in Unix milliseconds, or -1 while the file is still processing
	ExpirationTimestamp int64 `json:"expirationTimestamp"`
	// IsExpired reports whether the link has expired; the schema types it as a string, but every example sends null
	IsExpired bool `json:"isExpired"`
}

// UIncomeHistoryRequest holds the parameters of Get Income History
type UIncomeHistoryRequest struct {
	Symbol     currency.Pair
	IncomeType string // TRANSFER, REALIZED_PNL, FUNDING_FEE, COMMISSION and so on; empty for every type
	StartTime  time.Time
	EndTime    time.Time
	Page       uint64
	Limit      uint64
}

// UAccountIncomeHistory holds one income record from Get Income History
type UAccountIncomeHistory struct {
	Symbol        string        `json:"symbol"`
	IncomeType    string        `json:"incomeType"`
	Income        types.Number  `json:"income"`
	Asset         currency.Code `json:"asset"`
	Info          string        `json:"info"`
	Time          types.Time    `json:"time"`
	TransactionID uint64        `json:"tranId"`
	TradeID       string        `json:"tradeId"`
}

// UNotionalLeverageAndBrackets holds a symbol's notional and leverage brackets
type UNotionalLeverageAndBrackets struct {
	Symbol string `json:"symbol"`
	// NotionalCoefficient is the user's bracket multiplier, only sent when the symbol's brackets were adjusted
	NotionalCoefficient float64            `json:"notionalCoef"`
	Brackets            []UNotionalBracket `json:"brackets"`
}

// UNotionalBracket holds one notional and leverage bracket
type UNotionalBracket struct {
	Bracket                uint64  `json:"bracket"`
	InitialLeverage        uint64  `json:"initialLeverage"`
	NotionalCap            float64 `json:"notionalCap"`
	NotionalFloor          float64 `json:"notionalFloor"`
	MaintenanceMarginRatio float64 `json:"maintMarginRatio"`
	Cumulative             float64 `json:"cum"` // auxiliary number for quick calculation
}

// RateLimitInfo holds a rate limit
type RateLimitInfo struct {
	RateLimitType string `json:"rateLimitType"`
	Interval      string `json:"interval"`
	IntervalNum   int64  `json:"intervalNum"`
	Limit         int64  `json:"limit"`
}

// UCommissionRateResponse holds the reply of User Commission Rate
type UCommissionRateResponse struct {
	Symbol              string       `json:"symbol"`
	MakerCommissionRate types.Number `json:"makerCommissionRate"`
	TakerCommissionRate types.Number `json:"takerCommissionRate"`
	RPICommissionRate   types.Number `json:"rpiCommissionRate"`
}

// UBasisRequest holds the parameters of Basis
type UBasisRequest struct {
	Pair         currency.Pair
	ContractType string // PERPETUAL, CURRENT_QUARTER or NEXT_QUARTER
	Period       string // 5m, 15m, 30m, 1h, 2h, 4h, 6h, 12h or 1d
	StartTime    time.Time
	EndTime      time.Time
	Limit        uint64
}

// BasisInfo holds a period's basis between the index price and a futures contract's price
type BasisInfo struct {
	IndexPrice          types.Number `json:"indexPrice"`
	ContractType        string       `json:"contractType"`
	BasisRate           types.Number `json:"basisRate"`
	FuturesPrice        types.Number `json:"futuresPrice"`
	AnnualizedBasisRate types.Number `json:"annualizedBasisRate"` // empty for perpetual contracts
	Basis               types.Number `json:"basis"`
	Pair                string       `json:"pair"`
	Timestamp           types.Time   `json:"timestamp"`
}

// UServerTimeResponse holds the reply of Check Server Time
type UServerTimeResponse struct {
	ServerTime types.Time `json:"serverTime"`
}

// UCompositeIndex holds a composite index symbol's components
type UCompositeIndex struct {
	Symbol        string                     `json:"symbol"`
	Time          types.Time                 `json:"time"`
	Component     string                     `json:"component"`
	BaseAssetList []UCompositeIndexBaseAsset `json:"baseAssetList"`
}

// UCompositeIndexBaseAsset holds one component of a composite index
type UCompositeIndexBaseAsset struct {
	BaseAsset          currency.Code `json:"baseAsset"`
	QuoteAsset         currency.Code `json:"quoteAsset"`
	WeightInQuantity   types.Number  `json:"weightInQuantity"`
	WeightInPercentage types.Number  `json:"weightInPercentage"`
}

// UCompressedTradesRequest holds the parameters of Compressed/Aggregate Trades List
type UCompressedTradesRequest struct {
	Symbol    currency.Pair
	FromID    uint64
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
}

// UCompressedTradeData holds an aggregate trade
type UCompressedTradeData struct {
	AggregateTradeID uint64       `json:"a"`
	Price            types.Number `json:"p"`
	Quantity         types.Number `json:"q"`
	// NormalQuantity is the aggregate quantity excluding trades that involved retail price improvement orders
	NormalQuantity types.Number `json:"nq"`
	FirstTradeID   uint64       `json:"f"`
	LastTradeID    uint64       `json:"l"`
	Timestamp      types.Time   `json:"T"`
	IsBuyerMaker   bool         `json:"m"`
}

// UContinuousKlineRequest holds the parameters of Continuous Contract Kline/Candlestick Data
type UContinuousKlineRequest struct {
	Pair         currency.Pair
	ContractType string // PERPETUAL, CURRENT_QUARTER, NEXT_QUARTER or TRADIFI_PERPETUAL
	Interval     string
	StartTime    time.Time
	EndTime      time.Time
	Limit        uint64
}

// UKline holds a kline from Kline/Candlestick Data or Continuous Contract Kline/Candlestick Data
type UKline struct {
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

// UnmarshalJSON decodes a kline from its array form, whose last element is documented as ignore
func (k *UKline) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[12]any{&k.OpenTime, &k.Open, &k.High, &k.Low, &k.Close, &k.Volume, &k.CloseTime, &k.QuoteAssetVolume, &k.NumberOfTrades, &k.TakerBuyBaseAssetVolume, &k.TakerBuyQuoteAssetVolume, nil})
}

// UExchangeInfoResponse holds the reply of Exchange Information
type UExchangeInfoResponse struct {
	Timezone        string               `json:"timezone"`
	ServerTime      types.Time           `json:"serverTime"`
	FuturesType     string               `json:"futuresType"`
	RateLimits      []RateLimitInfo      `json:"rateLimits"`
	ExchangeFilters []string             `json:"exchangeFilters"`
	Assets          []UExchangeInfoAsset `json:"assets"`
	Symbols         []UFuturesSymbolInfo `json:"symbols"`
}

// UExchangeInfoAsset holds a margin asset's Multi-Assets mode settings
type UExchangeInfoAsset struct {
	Asset             currency.Code `json:"asset"`
	MarginAvailable   bool          `json:"marginAvailable"`
	AutoAssetExchange types.Number  `json:"autoAssetExchange"` // auto-exchange threshold in Multi-Assets mode
}

// UFuturesSymbolInfo holds a USDⓈ-M futures symbol's trading rules
type UFuturesSymbolInfo struct {
	Symbol                   string                 `json:"symbol"`
	Pair                     string                 `json:"pair"`
	ContractType             string                 `json:"contractType"`
	DeliveryDate             types.Time             `json:"deliveryDate"`
	OnboardDate              types.Time             `json:"onboardDate"`
	Status                   string                 `json:"status"`
	MaintenanceMarginPercent types.Number           `json:"maintMarginPercent"`
	RequiredMarginPercent    types.Number           `json:"requiredMarginPercent"`
	BaseAsset                currency.Code          `json:"baseAsset"`
	QuoteAsset               currency.Code          `json:"quoteAsset"`
	MarginAsset              currency.Code          `json:"marginAsset"`
	PricePrecision           uint64                 `json:"pricePrecision"`
	QuantityPrecision        uint64                 `json:"quantityPrecision"`
	BaseAssetPrecision       uint64                 `json:"baseAssetPrecision"`
	QuotePrecision           uint64                 `json:"quotePrecision"`
	UnderlyingType           string                 `json:"underlyingType"`
	UnderlyingSubType        []string               `json:"underlyingSubType"`
	TriggerProtect           types.Number           `json:"triggerProtect"`
	LiquidationFee           types.Number           `json:"liquidationFee"`
	MarketTakeBound          types.Number           `json:"marketTakeBound"`
	MaxMoveOrderLimit        uint64                 `json:"maxMoveOrderLimit"`
	Filters                  []UFuturesSymbolFilter `json:"filters"`
	OrderTypes               []string               `json:"orderTypes"`
	TimeInForce              []string               `json:"timeInForce"`
	PermissionSets           []string               `json:"permissionSets"`
}

// UFuturesSymbolFilter holds a symbol trading filter; each filter type sends only its own fields
type UFuturesSymbolFilter struct {
	FilterType          string       `json:"filterType"`
	MinPrice            types.Number `json:"minPrice"`
	MaxPrice            types.Number `json:"maxPrice"`
	TickSize            types.Number `json:"tickSize"`
	MinQuantity         types.Number `json:"minQty"`
	MaxQuantity         types.Number `json:"maxQty"`
	StepSize            types.Number `json:"stepSize"`
	Limit               uint64       `json:"limit"`
	Notional            types.Number `json:"notional"`
	MultiplierUp        types.Number `json:"multiplierUp"`
	MultiplierDown      types.Number `json:"multiplierDown"`
	MultiplierDecimal   types.Number `json:"multiplierDecimal"`
	PositionControlSide string       `json:"positionControlSide"`
}

// FundingRateHistory holds a funding rate record from Get Funding Rate History
type FundingRateHistory struct {
	Symbol      string       `json:"symbol"`
	FundingRate types.Number `json:"fundingRate"`
	FundingTime types.Time   `json:"fundingTime"`
	MarkPrice   types.Number `json:"markPrice"`
	RateType    string       `json:"rateType"` // Regular, or Special for the additional rate stock dividends generate
}

// FundingRateInfoResponse holds a symbol's adjusted funding rate cap, floor and interval from Get Funding Rate Info
type FundingRateInfoResponse struct {
	Symbol                   string       `json:"symbol"`
	AdjustedFundingRateCap   types.Number `json:"adjustedFundingRateCap"`
	AdjustedFundingRateFloor types.Number `json:"adjustedFundingRateFloor"`
	FundingIntervalHours     uint64       `json:"fundingIntervalHours"`
	Disclaimer               bool         `json:"disclaimer"`
	UpdateTime               types.Time   `json:"updateTime"` // not in the schema, but sent live
}

// UIndexPriceKlineRequest holds the parameters of Index Price Kline/Candlestick Data
type UIndexPriceKlineRequest struct {
	Pair      currency.Pair
	Interval  string
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
}

// UPriceKline holds a kline from the index price, mark price and premium index kline endpoints, whose
// volume and trade count elements are documented as ignore
type UPriceKline struct {
	OpenTime  types.Time
	Open      types.Number
	High      types.Number
	Low       types.Number
	Close     types.Number
	CloseTime types.Time
}

// UnmarshalJSON decodes a price kline from its array form
func (k *UPriceKline) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[12]any{&k.OpenTime, &k.Open, &k.High, &k.Low, &k.Close, nil, &k.CloseTime, nil, nil, nil, nil, nil})
}

// UKlineRequest holds the parameters of Kline/Candlestick Data, Mark Price Kline/Candlestick Data and Premium Index
// Kline Data
type UKlineRequest struct {
	Symbol    currency.Pair
	Interval  string
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
}

// UFuturesDataRequest holds the parameters of the /futures/data statistics endpoints: Open Interest Statistics,
// Long/Short Ratio, Top Trader Long/Short Account and Position Ratios, and Taker Buy/Sell Volume
type UFuturesDataRequest struct {
	Symbol    currency.Pair
	Period    string // 5m, 15m, 30m, 1h, 2h, 4h, 6h, 12h or 1d
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
}

// ULongShortRatio holds a period's long/short ratio of accounts or positions
type ULongShortRatio struct {
	Symbol         string       `json:"symbol"`
	LongShortRatio types.Number `json:"longShortRatio"`
	// LongAccount and ShortAccount are account shares on the account ratio endpoints and position shares on Top
	// Trader Long/Short Position Ratio, which keeps the same field names
	LongAccount  types.Number `json:"longAccount"`
	ShortAccount types.Number `json:"shortAccount"`
	Timestamp    types.Time   `json:"timestamp"`
}

// UMarkPrice holds a symbol's mark price and funding rate
type UMarkPrice struct {
	Symbol               string       `json:"symbol"`
	MarkPrice            types.Number `json:"markPrice"`
	IndexPrice           types.Number `json:"indexPrice"`
	EstimatedSettlePrice types.Number `json:"estimatedSettlePrice"`
	LastFundingRate      types.Number `json:"lastFundingRate"`
	InterestRate         types.Number `json:"interestRate"`
	NextFundingTime      types.Time   `json:"nextFundingTime"`
	Time                 types.Time   `json:"time"`
}

// AssetIndex holds an asset's index price and Multi-Assets mode buffers and rates
type AssetIndex struct {
	Symbol                string       `json:"symbol"`
	Time                  types.Time   `json:"time"`
	Index                 types.Number `json:"index"`
	BidBuffer             types.Number `json:"bidBuffer"`
	AskBuffer             types.Number `json:"askBuffer"`
	BidRate               types.Number `json:"bidRate"`
	AskRate               types.Number `json:"askRate"`
	AutoExchangeBidBuffer types.Number `json:"autoExchangeBidBuffer"`
	AutoExchangeAskBuffer types.Number `json:"autoExchangeAskBuffer"`
	AutoExchangeBidRate   types.Number `json:"autoExchangeBidRate"`
	AutoExchangeAskRate   types.Number `json:"autoExchangeAskRate"`
}

// AssetIndexResponse holds the reply of Asset Index: one index when a symbol is requested, every index otherwise
type AssetIndexResponse []AssetIndex

// UnmarshalJSON decodes a single asset index or an array of them
func (a *AssetIndexResponse) UnmarshalJSON(data []byte) error {
	return (*objectOrArray[AssetIndex])(a).UnmarshalJSON(data)
}

// UPublicTradesData holds a market trade from Recent Trades List or Old Trades Lookup
type UPublicTradesData struct {
	ID            uint64       `json:"id"`
	Price         types.Number `json:"price"`
	Quantity      types.Number `json:"qty"`
	QuoteQuantity types.Number `json:"quoteQty"`
	Time          types.Time   `json:"time"`
	IsBuyerMaker  bool         `json:"isBuyerMaker"`
	IsRPITrade    bool         `json:"isRPITrade"`
}

// UOpenInterestResponse holds the reply of Open Interest
type UOpenInterestResponse struct {
	OpenInterest types.Number `json:"openInterest"`
	Symbol       string       `json:"symbol"`
	Time         types.Time   `json:"time"`
}

// UOpenInterestStats holds a period's open interest statistics
type UOpenInterestStats struct {
	Symbol               string       `json:"symbol"`
	SumOpenInterest      types.Number `json:"sumOpenInterest"`
	SumOpenInterestValue types.Number `json:"sumOpenInterestValue"`
	CMCCirculatingSupply types.Number `json:"CMCCirculatingSupply"` // circulating supply provided by CoinMarketCap
	Timestamp            types.Time   `json:"timestamp"`
}

// UOrderBookResponse holds the reply of Order Book and RPI Order Book
type UOrderBookResponse struct {
	LastUpdateID      uint64                           `json:"lastUpdateId"`
	MessageOutputTime types.Time                       `json:"E"`
	TransactionTime   types.Time                       `json:"T"`
	Bids              orderbook.LevelsArrayPriceAmount `json:"bids"`
	Asks              orderbook.LevelsArrayPriceAmount `json:"asks"`
}

// SettlementPrice holds a quarterly contract's settlement price
type SettlementPrice struct {
	DeliveryTime  types.Time `json:"deliveryTime"`
	DeliveryPrice float64    `json:"deliveryPrice"`
}

// UIndexPriceConstituentsResponse holds the reply of Query Index Price Constituents
type UIndexPriceConstituentsResponse struct {
	Symbol       string                   `json:"symbol"`
	Time         types.Time               `json:"time"`
	Constituents []UIndexPriceConstituent `json:"constituents"`
}

// UIndexPriceConstituent holds one exchange's contribution to an index price; TradFi perps report a price of -1
type UIndexPriceConstituent struct {
	Exchange string       `json:"exchange"`
	Symbol   string       `json:"symbol"`
	Price    types.Number `json:"price"`
	Weight   types.Number `json:"weight"`
}

// USymbolOrderbookTicker holds a symbol's best bid and ask
type USymbolOrderbookTicker struct {
	Symbol       string       `json:"symbol"`
	BidPrice     types.Number `json:"bidPrice"`
	BidQuantity  types.Number `json:"bidQty"`
	AskPrice     types.Number `json:"askPrice"`
	AskQuantity  types.Number `json:"askQty"`
	Time         types.Time   `json:"time"`
	LastUpdateID uint64       `json:"lastUpdateId"` // not in the schema, but sent live
}

// USymbolPriceTicker holds a symbol's latest price
type USymbolPriceTicker struct {
	Symbol string       `json:"symbol"`
	Price  types.Number `json:"price"`
	Time   types.Time   `json:"time"`
}

// UTakerVolumeData holds a period's taker buy and sell volumes
type UTakerVolumeData struct {
	BuySellRatio types.Number `json:"buySellRatio"`
	BuyVolume    types.Number `json:"buyVol"`
	SellVolume   types.Number `json:"sellVol"`
	Timestamp    types.Time   `json:"timestamp"`
}

// U24HourPriceChangeStats holds a symbol's 24 hour rolling window price change statistics
type U24HourPriceChangeStats struct {
	Symbol               string       `json:"symbol"`
	PriceChange          types.Number `json:"priceChange"`
	PriceChangePercent   types.Number `json:"priceChangePercent"`
	WeightedAveragePrice types.Number `json:"weightedAvgPrice"`
	LastPrice            types.Number `json:"lastPrice"`
	LastQuantity         types.Number `json:"lastQty"`
	OpenPrice            types.Number `json:"openPrice"`
	HighPrice            types.Number `json:"highPrice"`
	LowPrice             types.Number `json:"lowPrice"`
	Volume               types.Number `json:"volume"`
	QuoteVolume          types.Number `json:"quoteVolume"`
	OpenTime             types.Time   `json:"openTime"`
	CloseTime            types.Time   `json:"closeTime"`
	FirstID              uint64       `json:"firstId"`
	LastID               uint64       `json:"lastId"`
	Count                uint64       `json:"count"`
}

// UAccountTradeListRequest holds the parameters of Account Trade List
type UAccountTradeListRequest struct {
	Symbol    currency.Pair
	OrderID   uint64
	StartTime time.Time
	EndTime   time.Time
	FromID    uint64 // cannot be sent with StartTime or EndTime
	Limit     uint64
}

// UAccountTradeHistory holds one of the account's trades
type UAccountTradeHistory struct {
	Buyer           bool          `json:"buyer"`
	Commission      types.Number  `json:"commission"`
	CommissionAsset currency.Code `json:"commissionAsset"`
	ID              uint64        `json:"id"`
	Maker           bool          `json:"maker"`
	OrderID         uint64        `json:"orderId"`
	Price           types.Number  `json:"price"`
	Quantity        types.Number  `json:"qty"`
	// QuoteQuantity is populated for USDⓈ-M symbols and BaseQuantity for COIN-M symbols; the other one is 0
	QuoteQuantity types.Number  `json:"quoteQty"`
	BaseQuantity  types.Number  `json:"baseQty"`
	MarginAsset   currency.Code `json:"marginAsset"`
	RealizedPNL   types.Number  `json:"realizedPnl"`
	Side          string        `json:"side"`
	PositionSide  string        `json:"positionSide"`
	Symbol        string        `json:"symbol"`
	Pair          string        `json:"pair"`
	Time          types.Time    `json:"time"`
}

// UAllOrdersRequest holds the parameters of All Orders
type UAllOrdersRequest struct {
	Symbol    currency.Pair
	OrderID   uint64 // returns orders with this ID and newer ones
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
}

// UOrderBase holds the fields every USDⓈ-M order reply carries
type UOrderBase struct {
	ClientOrderID           string       `json:"clientOrderId"`
	ExecutedQuantity        types.Number `json:"executedQty"`
	OrderID                 uint64       `json:"orderId"`
	OriginalQuantity        types.Number `json:"origQty"`
	Price                   types.Number `json:"price"`
	ReduceOnly              bool         `json:"reduceOnly"`
	Side                    string       `json:"side"`
	PositionSide            string       `json:"positionSide"`
	Status                  string       `json:"status"`
	StopPrice               types.Number `json:"stopPrice"` // ignore when the order type is TRAILING_STOP_MARKET
	ClosePosition           bool         `json:"closePosition"`
	Symbol                  string       `json:"symbol"`
	TimeInForce             string       `json:"timeInForce"`
	Type                    string       `json:"type"`
	OriginalType            string       `json:"origType"`
	UpdateTime              types.Time   `json:"updateTime"`
	WorkingType             string       `json:"workingType"`
	PriceProtect            bool         `json:"priceProtect"`
	PriceMatch              string       `json:"priceMatch"`
	SelfTradePreventionMode string       `json:"selfTradePreventionMode"`
	GoodTillDate            types.Time   `json:"goodTillDate"`
}

// UOrderResponse holds an order's state from Query Order, Query Current Open Order and Current All Open Orders
type UOrderResponse struct {
	UOrderBase
	AveragePrice    types.Number `json:"avgPrice"`
	CumulativeQuote types.Number `json:"cumQuote"`
	Time            types.Time   `json:"time"`
	ActivatePrice   types.Number `json:"activatePrice"` // only sent for TRAILING_STOP_MARKET orders
	PriceRate       types.Number `json:"priceRate"`     // callback rate, only sent for TRAILING_STOP_MARKET orders
}

// UFuturesOrderData holds an order from All Orders, which may also list COIN-M orders
type UFuturesOrderData struct {
	UOrderResponse
	CumulativeBase types.Number `json:"cumBase"`
	Pair           string       `json:"pair"`
}

// UAutoCancelAllOpenOrdersResponse holds the reply of Auto-Cancel All Open Orders
type UAutoCancelAllOpenOrdersResponse struct {
	Symbol        string       `json:"symbol"`
	CountdownTime types.Number `json:"countdownTime"` // milliseconds
}

// UAlgoOrderBase holds the fields every USDⓈ-M algo (conditional) order reply carries
type UAlgoOrderBase struct {
	AlgoID                  uint64       `json:"algoId"`
	ClientAlgoID            string       `json:"clientAlgoId"`
	AlgoType                string       `json:"algoType"`
	OrderType               string       `json:"orderType"`
	Symbol                  string       `json:"symbol"`
	Side                    string       `json:"side"`
	PositionSide            string       `json:"positionSide"`
	TimeInForce             string       `json:"timeInForce"`
	Quantity                types.Number `json:"quantity"`
	AlgoStatus              string       `json:"algoStatus"`
	TriggerPrice            types.Number `json:"triggerPrice"`
	Price                   types.Number `json:"price"`
	IcebergQuantity         string       `json:"icebergQuantity"` // "null": algo orders take no iceberg quantity
	SelfTradePreventionMode string       `json:"selfTradePreventionMode"`
	WorkingType             string       `json:"workingType"`
	PriceMatch              string       `json:"priceMatch"`
	ClosePosition           bool         `json:"closePosition"`
	PriceProtect            bool         `json:"priceProtect"`
	ReduceOnly              bool         `json:"reduceOnly"`
	CreateTime              types.Time   `json:"createTime"`
	UpdateTime              types.Time   `json:"updateTime"`
	TriggerTime             types.Time   `json:"triggerTime"`
	GoodTillDate            types.Time   `json:"goodTillDate"`
}

// UAlgoOrderResponse holds an algo order's state from Query Algo Order
type UAlgoOrderResponse struct {
	UAlgoOrderBase
	ActualOrderID       string       `json:"actualOrderId"` // empty until the order triggers
	ActualPrice         types.Number `json:"actualPrice"`   // the average fill price once filled
	ActualType          string       `json:"actualType"`    // only sent once the order triggers
	ActualQuantity      types.Number `json:"actualQty"`     // only sent once the order fills
	TakeProfitOrderType string       `json:"tpOrderType"`
}

// UNewAlgoOrderRequest holds the parameters of New Algo Order
type UNewAlgoOrderRequest struct {
	AlgoType     string // CONDITIONAL
	Symbol       currency.Pair
	Side         string
	PositionSide string
	// OrderType is STOP_MARKET, TAKE_PROFIT_MARKET, STOP, TAKE_PROFIT or TRAILING_STOP_MARKET
	OrderType               string
	TimeInForce             string
	Quantity                float64 // cannot be sent with ClosePosition
	Price                   float64 // cannot be sent with PriceMatch
	TriggerPrice            float64
	WorkingType             string // MARK_PRICE or CONTRACT_PRICE
	PriceMatch              string
	ClosePosition           bool
	PriceProtect            bool
	ReduceOnly              bool // cannot be sent with ClosePosition
	ActivatePrice           float64
	CallbackRate            float64
	ClientAlgoID            string
	NewOrderRespType        string
	SelfTradePreventionMode string
	GoodTillDate            time.Time // required for GTD orders; Binance keeps second precision
}

// UNewAlgoOrderResponse holds the reply of New Algo Order
type UNewAlgoOrderResponse struct {
	UAlgoOrderBase
	ActivatePrice types.Number `json:"activatePrice"` // only set for TRAILING_STOP_MARKET orders
	CallbackRate  types.Number `json:"callbackRate"`  // only set for TRAILING_STOP_MARKET orders
}

// UCancelAlgoOrderResponse holds the reply of Cancel Algo Order
type UCancelAlgoOrderResponse struct {
	AlgoID       uint64       `json:"algoId"`
	ClientAlgoID string       `json:"clientAlgoId"`
	Code         types.Number `json:"code"`
	Message      string       `json:"msg"`
}

// UModifyOrderRequest holds the parameters of Modify Order and one order of Modify Multiple Orders
type UModifyOrderRequest struct {
	OrderID           uint64 // OrderID wins when both IDs are set
	OrigClientOrderID string
	Symbol            currency.Pair
	Side              string
	Quantity          float64
	Price             float64 // required unless PriceMatch is set, and cannot be sent with it
	PriceMatch        string
	ModifyID          uint64 // returned unchanged in the reply
	// ReduceOnly skips the minimum notional check when the original order is reduce-only, and is rejected otherwise
	ReduceOnly bool
}

// UNewOrderResponse holds the reply of New Order
type UNewOrderResponse struct {
	UOrderBase
	CumulativeQuantity types.Number `json:"cumQty"`
}

// UModifyOrderResponse holds the reply of Modify Order
type UModifyOrderResponse struct {
	UNewOrderResponse
	Pair     string `json:"pair"`
	ModifyID uint64 `json:"modifyId"` // only sent when the request set it
}

// UModifyBatchOrderResponse holds one order's reply from Modify Multiple Orders: the modified order, or the code and
// message that rejected it
type UModifyBatchOrderResponse struct {
	UModifyOrderResponse
	Code    int64  `json:"code"`
	Message string `json:"msg"`
}

// UFuturesNewOrderRequest holds the parameters of New Order and one order of Place Multiple Orders. Stop, take profit
// and trailing stop orders are placed with UNewAlgoOrder
type UFuturesNewOrderRequest struct {
	Symbol                  currency.Pair
	Side                    string
	PositionSide            string
	OrderType               string // LIMIT or MARKET
	TimeInForce             string
	ReduceOnly              bool
	Quantity                float64
	Price                   float64 // cannot be sent with PriceMatch
	NewClientOrderID        string
	NewOrderRespType        string
	PriceMatch              string
	SelfTradePreventionMode string
	GoodTillDate            time.Time // required for GTD orders; Binance keeps second precision
}

// UPlaceBatchOrderResponse holds one order's reply from Place Multiple Orders: the new order, or the code and message
// that rejected it
type UPlaceBatchOrderResponse struct {
	UNewOrderResponse
	Code    int64  `json:"code"`
	Message string `json:"msg"`
}

// UCancelOrderResponse holds the reply of Cancel Order
type UCancelOrderResponse struct {
	UNewOrderResponse
	ActivatePrice types.Number `json:"activatePrice"` // only sent for TRAILING_STOP_MARKET orders
	PriceRate     types.Number `json:"priceRate"`     // callback rate, only sent for TRAILING_STOP_MARKET orders
}

// UCancelBatchOrderResponse holds one order's reply from Cancel Multiple Orders: the cancelled order, or the code and
// message that rejected it
type UCancelBatchOrderResponse struct {
	UCancelOrderResponse
	Code    int64  `json:"code"`
	Message string `json:"msg"`
}

// UChangeInitialLeverageResponse holds the reply of Change Initial Leverage
type UChangeInitialLeverageResponse struct {
	Leverage         uint64       `json:"leverage"`
	MaxNotionalValue types.Number `json:"maxNotionalValue"`
	Symbol           string       `json:"symbol"`
}

// UAlgoOrder holds an algo order from Current All Algo Open Orders and Query All Algo Orders
type UAlgoOrder struct {
	UAlgoOrderBase
	ActualOrderID          string       `json:"actualOrderId"` // empty until the order triggers
	ActualPrice            types.Number `json:"actualPrice"`   // the average fill price once filled
	TakeProfitTriggerPrice types.Number `json:"tpTriggerPrice"`
	TakeProfitPrice        types.Number `json:"tpPrice"`
	StopLossTriggerPrice   types.Number `json:"slTriggerPrice"`
	StopLossPrice          types.Number `json:"slPrice"`
	TakeProfitOrderType    string       `json:"tpOrderType"`
}

// UOrderModifyHistoryRequest holds the parameters of Get Order Modify History
type UOrderModifyHistoryRequest struct {
	Symbol            currency.Pair
	OrderID           uint64 // OrderID wins when both IDs are set
	OrigClientOrderID string
	StartTime         time.Time
	EndTime           time.Time
	Limit             uint64
}

// FuturesOrderAmendment holds one modification from Get Order Modify History
type FuturesOrderAmendment struct {
	AmendmentID   uint64         `json:"amendmentId"`
	Symbol        string         `json:"symbol"`
	Pair          string         `json:"pair"`
	OrderID       uint64         `json:"orderId"`
	ClientOrderID string         `json:"clientOrderId"`
	Time          types.Time     `json:"time"`
	Amendment     OrderAmendment `json:"amendment"`
}

// OrderAmendment holds what one modification changed
type OrderAmendment struct {
	Price            AmendmentChange `json:"price"`
	OriginalQuantity AmendmentChange `json:"origQty"`
	Count            uint64          `json:"count"`    // the number of times the order has been modified
	ModifyID         uint64          `json:"modifyId"` // only sent when the modify request set it
}

// AmendmentChange holds a value before and after a modification
type AmendmentChange struct {
	Before types.Number `json:"before"`
	After  types.Number `json:"after"`
}

// UPositionMarginChangeHistoryRequest holds the parameters of Get Position Margin Change History
type UPositionMarginChangeHistoryRequest struct {
	Symbol     currency.Pair
	ChangeType string // add or reduce; empty for both
	StartTime  time.Time
	EndTime    time.Time
	Limit      uint64
}

// UPositionMarginChange holds one isolated position margin change
type UPositionMarginChange struct {
	Symbol       string        `json:"symbol"`
	Type         uint64        `json:"type"` // 1 adds and 2 reduces position margin
	DeltaType    string        `json:"deltaType"`
	Amount       types.Number  `json:"amount"`
	Asset        currency.Code `json:"asset"`
	Time         types.Time    `json:"time"`
	PositionSide string        `json:"positionSide"`
}

// UModifyIsolatedPositionMarginResponse holds the reply of Modify Isolated Position Margin
type UModifyIsolatedPositionMarginResponse struct {
	Amount  float64 `json:"amount"`
	Code    int64   `json:"code"`
	Message string  `json:"msg"`
	Type    uint64  `json:"type"` // 1 adds and 2 reduces position margin
}

// UPositionADLQuantile holds a symbol's auto-deleveraging queue estimates
type UPositionADLQuantile struct {
	Symbol      string       `json:"symbol"`
	ADLQuantile UADLQuantile `json:"adlQuantile"`
}

// UADLQuantile holds the auto-deleveraging quantile, 0 (lowest) to 4 (highest), of each position side
type UADLQuantile struct {
	Long  uint64 `json:"LONG"`
	Short uint64 `json:"SHORT"`
	// Hedge is only a sign that cross positions in Hedge Mode share one quantile, sent instead of Both; ignore its value
	Hedge uint64 `json:"HEDGE"`
	Both  uint64 `json:"BOTH"`
}

// UPositionInformationV2 holds a position from Position Information V2
type UPositionInformationV2 struct {
	Symbol           string        `json:"symbol"`
	PositionAmount   types.Number  `json:"positionAmt"`
	EntryPrice       types.Number  `json:"entryPrice"`
	BreakEvenPrice   types.Number  `json:"breakEvenPrice"`
	MarkPrice        types.Number  `json:"markPrice"`
	UnrealizedProfit types.Number  `json:"unRealizedProfit"`
	LiquidationPrice types.Number  `json:"liquidationPrice"`
	Leverage         types.Number  `json:"leverage"`
	MaxNotionalValue types.Number  `json:"maxNotionalValue"`
	MarginType       string        `json:"marginType"`
	IsolatedMargin   types.Number  `json:"isolatedMargin"`
	IsAutoAddMargin  types.Boolean `json:"isAutoAddMargin"`
	PositionSide     string        `json:"positionSide"`
	Notional         types.Number  `json:"notional"`
	IsolatedWallet   types.Number  `json:"isolatedWallet"`
	UpdateTime       types.Time    `json:"updateTime"`
}

// UPositionInformationV3 holds a position from Position Information V3, which only lists symbols with a position or
// open orders and leaves leverage and margin type to Symbol Configuration
type UPositionInformationV3 struct {
	Symbol                 string        `json:"symbol"`
	PositionSide           string        `json:"positionSide"`
	PositionAmount         types.Number  `json:"positionAmt"`
	EntryPrice             types.Number  `json:"entryPrice"`
	BreakEvenPrice         types.Number  `json:"breakEvenPrice"`
	MarkPrice              types.Number  `json:"markPrice"`
	UnrealizedProfit       types.Number  `json:"unRealizedProfit"`
	LiquidationPrice       types.Number  `json:"liquidationPrice"`
	IsolatedMargin         types.Number  `json:"isolatedMargin"`
	Notional               types.Number  `json:"notional"`
	MarginAsset            currency.Code `json:"marginAsset"`
	IsolatedWallet         types.Number  `json:"isolatedWallet"`
	InitialMargin          types.Number  `json:"initialMargin"`
	MaintenanceMargin      types.Number  `json:"maintMargin"`
	PositionInitialMargin  types.Number  `json:"positionInitialMargin"`
	OpenOrderInitialMargin types.Number  `json:"openOrderInitialMargin"`
	ADL                    uint64        `json:"adl"` // auto-deleveraging ranking
	BidNotional            types.Number  `json:"bidNotional"`
	AskNotional            types.Number  `json:"askNotional"`
	UpdateTime             types.Time    `json:"updateTime"`
}

// UAllAlgoOrdersRequest holds the parameters of Query All Algo Orders
type UAllAlgoOrdersRequest struct {
	Symbol    currency.Pair
	AlgoID    uint64 // returns algo orders with this ID and newer ones
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
}

// UForceOrdersRequest holds the parameters of User's Force Orders
type UForceOrdersRequest struct {
	Symbol        currency.Pair
	AutoCloseType string // LIQUIDATION or ADL; empty for both
	StartTime     time.Time
	EndTime       time.Time
	Limit         uint64
}

// UForceOrder holds a liquidation or auto-deleveraging order from User's Force Orders, which may also list COIN-M
// orders
type UForceOrder struct {
	OrderID          uint64       `json:"orderId"`
	Symbol           string       `json:"symbol"`
	Pair             string       `json:"pair"`
	Status           string       `json:"status"`
	ClientOrderID    string       `json:"clientOrderId"`
	Price            types.Number `json:"price"`
	AveragePrice     types.Number `json:"avgPrice"`
	OriginalQuantity types.Number `json:"origQty"`
	ExecutedQuantity types.Number `json:"executedQty"`
	CumulativeQuote  types.Number `json:"cumQuote"`
	CumulativeBase   types.Number `json:"cumBase"`
	TimeInForce      string       `json:"timeInForce"`
	Type             string       `json:"type"`
	ReduceOnly       bool         `json:"reduceOnly"`
	ClosePosition    bool         `json:"closePosition"`
	Side             string       `json:"side"`
	PositionSide     string       `json:"positionSide"`
	StopPrice        types.Number `json:"stopPrice"`
	WorkingType      string       `json:"workingType"`
	OriginalType     string       `json:"origType"`
	Time             types.Time   `json:"time"`
	UpdateTime       types.Time   `json:"updateTime"`
}

// FuturesListenKeyResponse holds the listen key of a USDⓈ-M or COIN-M user data stream
type FuturesListenKeyResponse struct {
	ListenKey string `json:"listenKey"`
}
