package binance

import (
	"errors"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/types"
)

var (
	errContractTypeIsRequired  = errors.New("contract type is required")
	errInvalidPeriodOrInterval = errors.New("invalid period")
)

// USDTMarginedFuturesAccountInformationV2Response holds the reply of Account Information V2
type USDTMarginedFuturesAccountInformationV2Response struct {
	FeeTier                     uint64                        `json:"feeTier"`
	FeeBurn                     bool                          `json:"feeBurn"`
	CanTrade                    bool                          `json:"canTrade"`
	CanDeposit                  bool                          `json:"canDeposit"`
	CanWithdraw                 bool                          `json:"canWithdraw"`
	UpdateTime                  types.Time                    `json:"updateTime"`
	MultiAssetsMargin           bool                          `json:"multiAssetsMargin"`
	TradeGroupID                int64                         `json:"tradeGroupId"` // -1 when the account is in no trade group
	TotalInitialMargin          types.Number                  `json:"totalInitialMargin"`
	TotalMaintenanceMargin      types.Number                  `json:"totalMaintMargin"`
	TotalWalletBalance          types.Number                  `json:"totalWalletBalance"`
	TotalUnrealisedProfit       types.Number                  `json:"totalUnrealizedProfit"`
	TotalMarginBalance          types.Number                  `json:"totalMarginBalance"`
	TotalPositionInitialMargin  types.Number                  `json:"totalPositionInitialMargin"`
	TotalOpenOrderInitialMargin types.Number                  `json:"totalOpenOrderInitialMargin"`
	TotalCrossWalletBalance     types.Number                  `json:"totalCrossWalletBalance"`
	TotalCrossUnrealisedPNL     types.Number                  `json:"totalCrossUnPnl"`
	AvailableBalance            types.Number                  `json:"availableBalance"`
	MaxWithdrawAmount           types.Number                  `json:"maxWithdrawAmount"`
	Assets                      []USDTMarginedFuturesAsset    `json:"assets"`
	Positions                   []USDTMarginedFuturesPosition `json:"positions"`
}

// USDTMarginedFuturesAsset holds an asset's balances in Account Information V2
type USDTMarginedFuturesAsset struct {
	Asset                  currency.Code `json:"asset"`
	WalletBalance          types.Number  `json:"walletBalance"`
	UnrealisedProfit       types.Number  `json:"unrealizedProfit"`
	MarginBalance          types.Number  `json:"marginBalance"`
	MaintenanceMargin      types.Number  `json:"maintMargin"`
	InitialMargin          types.Number  `json:"initialMargin"`
	PositionInitialMargin  types.Number  `json:"positionInitialMargin"`
	OpenOrderInitialMargin types.Number  `json:"openOrderInitialMargin"`
	CrossWalletBalance     types.Number  `json:"crossWalletBalance"`
	CrossUnrealisedPNL     types.Number  `json:"crossUnPnl"`
	AvailableBalance       types.Number  `json:"availableBalance"`
	MaxWithdrawAmount      types.Number  `json:"maxWithdrawAmount"`
	MarginAvailable        bool          `json:"marginAvailable"`
	UpdateTime             types.Time    `json:"updateTime"`
}

// USDTMarginedFuturesPosition holds a symbol's position in Account Information V2
type USDTMarginedFuturesPosition struct {
	Symbol                 string       `json:"symbol"`
	InitialMargin          types.Number `json:"initialMargin"`
	MaintenanceMargin      types.Number `json:"maintMargin"`
	UnrealisedProfit       types.Number `json:"unrealizedProfit"`
	PositionInitialMargin  types.Number `json:"positionInitialMargin"`
	OpenOrderInitialMargin types.Number `json:"openOrderInitialMargin"`
	Leverage               types.Number `json:"leverage"`
	Isolated               bool         `json:"isolated"`
	EntryPrice             types.Number `json:"entryPrice"`
	MaxNotional            types.Number `json:"maxNotional"`
	BidNotional            types.Number `json:"bidNotional"`
	AskNotional            types.Number `json:"askNotional"`
	PositionSide           PositionSide `json:"positionSide"`
	PositionAmount         types.Number `json:"positionAmt"`
	UpdateTime             types.Time   `json:"updateTime"`
}

// USDTMarginedFuturesAccountInformationV3Response holds the reply of Account Information V3, which leaves the account
// and symbol configuration to Futures Account Configuration and Symbol Configuration
type USDTMarginedFuturesAccountInformationV3Response struct {
	TotalInitialMargin          types.Number                    `json:"totalInitialMargin"`
	TotalMaintenanceMargin      types.Number                    `json:"totalMaintMargin"`
	TotalWalletBalance          types.Number                    `json:"totalWalletBalance"`
	TotalUnrealisedProfit       types.Number                    `json:"totalUnrealizedProfit"`
	TotalMarginBalance          types.Number                    `json:"totalMarginBalance"`
	TotalPositionInitialMargin  types.Number                    `json:"totalPositionInitialMargin"`
	TotalOpenOrderInitialMargin types.Number                    `json:"totalOpenOrderInitialMargin"`
	TotalCrossWalletBalance     types.Number                    `json:"totalCrossWalletBalance"`
	TotalCrossUnrealisedPNL     types.Number                    `json:"totalCrossUnPnl"`
	AvailableBalance            types.Number                    `json:"availableBalance"`
	MaxWithdrawAmount           types.Number                    `json:"maxWithdrawAmount"`
	Assets                      []USDTMarginedFuturesAssetV3    `json:"assets"`
	Positions                   []USDTMarginedFuturesPositionV3 `json:"positions"`
}

// USDTMarginedFuturesAssetV3 holds an asset's balances in Account Information V3
type USDTMarginedFuturesAssetV3 struct {
	Asset                  currency.Code `json:"asset"`
	WalletBalance          types.Number  `json:"walletBalance"`
	UnrealisedProfit       types.Number  `json:"unrealizedProfit"`
	MarginBalance          types.Number  `json:"marginBalance"`
	MaintenanceMargin      types.Number  `json:"maintMargin"`
	InitialMargin          types.Number  `json:"initialMargin"`
	PositionInitialMargin  types.Number  `json:"positionInitialMargin"`
	OpenOrderInitialMargin types.Number  `json:"openOrderInitialMargin"`
	CrossWalletBalance     types.Number  `json:"crossWalletBalance"`
	CrossUnrealisedPNL     types.Number  `json:"crossUnPnl"`
	AvailableBalance       types.Number  `json:"availableBalance"`
	MaxWithdrawAmount      types.Number  `json:"maxWithdrawAmount"`
	UpdateTime             types.Time    `json:"updateTime"`
}

// USDTMarginedFuturesPositionV3 holds a position in Account Information V3, which only lists symbols with a position or
// open orders
type USDTMarginedFuturesPositionV3 struct {
	Symbol            string       `json:"symbol"`
	PositionSide      PositionSide `json:"positionSide"`
	PositionAmount    types.Number `json:"positionAmt"`
	UnrealisedProfit  types.Number `json:"unrealizedProfit"`
	IsolatedMargin    types.Number `json:"isolatedMargin"`
	Notional          types.Number `json:"notional"`
	IsolatedWallet    types.Number `json:"isolatedWallet"`
	InitialMargin     types.Number `json:"initialMargin"`
	MaintenanceMargin types.Number `json:"maintMargin"`
	UpdateTime        types.Time   `json:"updateTime"`
}

// USDTMarginedFuturesAccountBalanceResponse holds an asset's balance from Futures Account Balance V3
type USDTMarginedFuturesAccountBalanceResponse struct {
	AccountAlias       string        `json:"accountAlias"`
	Asset              currency.Code `json:"asset"`
	Balance            types.Number  `json:"balance"`
	CrossWalletBalance types.Number  `json:"crossWalletBalance"`
	CrossUnrealisedPNL types.Number  `json:"crossUnPnl"`
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

// USDTMarginedFuturesMultiAssetsModeResponse holds the reply of Get Current Multi-Assets Mode
type USDTMarginedFuturesMultiAssetsModeResponse struct {
	MultiAssetsMargin bool `json:"multiAssetsMargin"` // true for Multi-Assets Mode, false for Single-Asset Mode
}

// USDTMarginedFuturesPositionModeResponse holds the reply of Get Current Position Mode
type USDTMarginedFuturesPositionModeResponse struct {
	DualSidePosition bool `json:"dualSidePosition"` // true for Hedge Mode, false for One-way Mode
}

// USDTMarginedFuturesDownloadIDResponse holds the reply of the Get Download Id endpoints for order, trade and
// transaction history
type USDTMarginedFuturesDownloadIDResponse struct {
	AverageDownloadMillisecondsOfLast30Days uint64 `json:"avgCostTimestampOfLast30d"`
	DownloadID                              string `json:"downloadId"`
}

// USDTMarginedFuturesDownloadLinkResponse holds the reply of the Download Link by Id endpoints for order, trade and
// transaction history
type USDTMarginedFuturesDownloadLinkResponse struct {
	DownloadID string `json:"downloadId"`
	Status     string `json:"status"` // completed or processing
	URL        string `json:"url"`
	Notified   bool   `json:"notified"`
	// ExpirationTimestamp is when the link expires in Unix milliseconds, or -1 while the file is still processing
	ExpirationTimestamp int64 `json:"expirationTimestamp"`
	// IsExpired reports whether the link has expired; the schema types it as a string, but every example sends null
	IsExpired bool `json:"isExpired"`
}

// USDTMarginedFuturesIncomeHistoryRequest holds the parameters of Get Income History
type USDTMarginedFuturesIncomeHistoryRequest struct {
	Symbol     currency.Pair
	IncomeType IncomeType // TRANSFER, REALIZED_PNL, FUNDING_FEE, COMMISSION and so on; empty for every type
	StartTime  time.Time
	EndTime    time.Time
	Page       uint64
	Limit      uint64
}

// USDTMarginedFuturesAccountIncomeHistoryResponse holds one income record from Get Income History
type USDTMarginedFuturesAccountIncomeHistoryResponse struct {
	Symbol        string        `json:"symbol"`
	IncomeType    IncomeType    `json:"incomeType"`
	Income        types.Number  `json:"income"`
	Asset         currency.Code `json:"asset"`
	Info          string        `json:"info"`
	Time          types.Time    `json:"time"`
	TransactionID uint64        `json:"tranId"`
	TradeID       string        `json:"tradeId"`
}

// USDTMarginedFuturesNotionalLeverageAndBracketsResponse holds a symbol's notional and leverage brackets
type USDTMarginedFuturesNotionalLeverageAndBracketsResponse struct {
	Symbol string `json:"symbol"`
	// NotionalCoefficient is the user's bracket multiplier, only sent when the symbol's brackets were adjusted
	NotionalCoefficient float64                              `json:"notionalCoef"`
	Brackets            []USDTMarginedFuturesNotionalBracket `json:"brackets"`
}

// USDTMarginedFuturesNotionalBracket holds one notional and leverage bracket
type USDTMarginedFuturesNotionalBracket struct {
	Bracket                uint64  `json:"bracket"`
	InitialLeverage        uint64  `json:"initialLeverage"`
	NotionalCap            float64 `json:"notionalCap"`
	NotionalFloor          float64 `json:"notionalFloor"`
	MaintenanceMarginRatio float64 `json:"maintMarginRatio"`
	Cumulative             float64 `json:"cum"` // auxiliary number for quick calculation
}

// RateLimitInfoResponse holds a rate limit
type RateLimitInfoResponse struct {
	RateLimitType  string `json:"rateLimitType"`
	Interval       string `json:"interval"`
	IntervalNumber uint64 `json:"intervalNum"`
	Limit          uint64 `json:"limit"`
}

// USDTMarginedFuturesCommissionRateResponse holds the reply of User Commission Rate
type USDTMarginedFuturesCommissionRateResponse struct {
	Symbol                               string       `json:"symbol"`
	MakerCommissionRate                  types.Number `json:"makerCommissionRate"`
	TakerCommissionRate                  types.Number `json:"takerCommissionRate"`
	RetailPriceImprovementCommissionRate types.Number `json:"rpiCommissionRate"`
}

// USDTMarginedFuturesBasisRequest holds the parameters of Basis
type USDTMarginedFuturesBasisRequest struct {
	Pair         currency.Pair
	ContractType ContractType   // PERPETUAL, CURRENT_QUARTER or NEXT_QUARTER
	Period       kline.Interval // 5m, 15m, 30m, 1h, 2h, 4h, 6h, 12h or 1d
	StartTime    time.Time
	EndTime      time.Time
	Limit        uint64
}

// BasisInfoResponse holds a period's basis between the index price and a futures contract's price
type BasisInfoResponse struct {
	IndexPrice          types.Number `json:"indexPrice"`
	ContractType        ContractType `json:"contractType"`
	BasisRate           types.Number `json:"basisRate"`
	FuturesPrice        types.Number `json:"futuresPrice"`
	AnnualisedBasisRate types.Number `json:"annualizedBasisRate"` // empty for perpetual contracts
	Basis               types.Number `json:"basis"`
	Pair                string       `json:"pair"`
	Timestamp           types.Time   `json:"timestamp"`
}

// USDTMarginedFuturesServerTimeResponse holds the reply of Check Server Time
type USDTMarginedFuturesServerTimeResponse struct {
	ServerTime types.Time `json:"serverTime"`
}

// USDTMarginedFuturesCompositeIndexResponse holds a composite index symbol's components
type USDTMarginedFuturesCompositeIndexResponse struct {
	Symbol        string                                       `json:"symbol"`
	Time          types.Time                                   `json:"time"`
	Component     string                                       `json:"component"`
	BaseAssetList []USDTMarginedFuturesCompositeIndexBaseAsset `json:"baseAssetList"`
}

// USDTMarginedFuturesCompositeIndexBaseAsset holds one component of a composite index
type USDTMarginedFuturesCompositeIndexBaseAsset struct {
	BaseAsset          currency.Code `json:"baseAsset"`
	QuoteAsset         currency.Code `json:"quoteAsset"`
	WeightInQuantity   types.Number  `json:"weightInQuantity"`
	WeightInPercentage types.Number  `json:"weightInPercentage"`
}

// USDTMarginedFuturesCompressedTradesRequest holds the parameters of Compressed/Aggregate Trades List
type USDTMarginedFuturesCompressedTradesRequest struct {
	Symbol    currency.Pair
	FromID    uint64
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
}

// USDTMarginedFuturesCompressedTradeResponse holds an aggregate trade
type USDTMarginedFuturesCompressedTradeResponse struct {
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

// USDTMarginedFuturesContinuousKlineRequest holds the parameters of Continuous Contract Kline/Candlestick Data
type USDTMarginedFuturesContinuousKlineRequest struct {
	Pair         currency.Pair
	ContractType ContractType // PERPETUAL, CURRENT_QUARTER, NEXT_QUARTER or TRADIFI_PERPETUAL
	Interval     kline.Interval
	StartTime    time.Time
	EndTime      time.Time
	Limit        uint64
}

// USDTMarginedFuturesKlineResponse holds a kline from Kline/Candlestick Data or Continuous Contract Kline/Candlestick
// Data
type USDTMarginedFuturesKlineResponse struct {
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
func (k *USDTMarginedFuturesKlineResponse) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[12]any{&k.OpenTime, &k.Open, &k.High, &k.Low, &k.Close, &k.Volume, &k.CloseTime, &k.QuoteAssetVolume, &k.NumberOfTrades, &k.TakerBuyBaseAssetVolume, &k.TakerBuyQuoteAssetVolume, nil})
}

// USDTMarginedFuturesExchangeInfoResponse holds the reply of Exchange Information
type USDTMarginedFuturesExchangeInfoResponse struct {
	Timezone        string                                 `json:"timezone"`
	ServerTime      types.Time                             `json:"serverTime"`
	FuturesType     string                                 `json:"futuresType"`
	RateLimits      []RateLimitInfoResponse                `json:"rateLimits"`
	ExchangeFilters []string                               `json:"exchangeFilters"`
	Assets          []USDTMarginedFuturesExchangeInfoAsset `json:"assets"`
	Symbols         []USDTMarginedFuturesSymbolInfo        `json:"symbols"`
}

// USDTMarginedFuturesExchangeInfoAsset holds a margin asset's Multi-Assets mode settings
type USDTMarginedFuturesExchangeInfoAsset struct {
	Asset             currency.Code `json:"asset"`
	MarginAvailable   bool          `json:"marginAvailable"`
	AutoAssetExchange types.Number  `json:"autoAssetExchange"` // auto-exchange threshold in Multi-Assets mode
}

// USDTMarginedFuturesSymbolInfo holds a USDⓈ-M futures symbol's trading rules
type USDTMarginedFuturesSymbolInfo struct {
	Symbol                   string                            `json:"symbol"`
	Pair                     string                            `json:"pair"`
	ContractType             ContractType                      `json:"contractType"`
	DeliveryDate             types.Time                        `json:"deliveryDate"`
	OnboardDate              types.Time                        `json:"onboardDate"`
	Status                   string                            `json:"status"`
	MaintenanceMarginPercent types.Number                      `json:"maintMarginPercent"`
	RequiredMarginPercent    types.Number                      `json:"requiredMarginPercent"`
	BaseAsset                currency.Code                     `json:"baseAsset"`
	QuoteAsset               currency.Code                     `json:"quoteAsset"`
	MarginAsset              currency.Code                     `json:"marginAsset"`
	PricePrecision           uint64                            `json:"pricePrecision"`
	QuantityPrecision        uint64                            `json:"quantityPrecision"`
	BaseAssetPrecision       uint64                            `json:"baseAssetPrecision"`
	QuotePrecision           uint64                            `json:"quotePrecision"`
	UnderlyingType           string                            `json:"underlyingType"`
	UnderlyingSubType        []string                          `json:"underlyingSubType"`
	TriggerProtect           types.Number                      `json:"triggerProtect"`
	LiquidationFee           types.Number                      `json:"liquidationFee"`
	MarketTakeBound          types.Number                      `json:"marketTakeBound"`
	MaxMoveOrderLimit        uint64                            `json:"maxMoveOrderLimit"`
	Filters                  []USDTMarginedFuturesSymbolFilter `json:"filters"`
	OrderTypes               []OrderType                       `json:"orderTypes"`
	TimeInForce              []TimeInForce                     `json:"timeInForce"`
	PermissionSets           []string                          `json:"permissionSets"`
}

// USDTMarginedFuturesSymbolFilter holds a symbol trading filter; each filter type sends only its own fields
type USDTMarginedFuturesSymbolFilter struct {
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

// FundingRateHistoryResponse holds a funding rate record from Get Funding Rate History
type FundingRateHistoryResponse struct {
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

// USDTMarginedFuturesIndexPriceKlineRequest holds the parameters of Index Price Kline/Candlestick Data
type USDTMarginedFuturesIndexPriceKlineRequest struct {
	Pair      currency.Pair
	Interval  kline.Interval
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
}

// USDTMarginedFuturesPriceKlineResponse holds a kline from the index price, mark price and premium index kline
// endpoints, whose volume and trade count elements are documented as ignore
type USDTMarginedFuturesPriceKlineResponse struct {
	OpenTime  types.Time
	Open      types.Number
	High      types.Number
	Low       types.Number
	Close     types.Number
	CloseTime types.Time
}

// UnmarshalJSON decodes a price kline from its array form
func (k *USDTMarginedFuturesPriceKlineResponse) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[12]any{&k.OpenTime, &k.Open, &k.High, &k.Low, &k.Close, nil, &k.CloseTime, nil, nil, nil, nil, nil})
}

// USDTMarginedFuturesKlineRequest holds the parameters of Kline/Candlestick Data, Mark Price Kline/Candlestick Data and
// Premium Index Kline Data
type USDTMarginedFuturesKlineRequest struct {
	Symbol    currency.Pair
	Interval  kline.Interval
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
}

// USDTMarginedFuturesDataRequest holds the parameters of the /futures/data statistics endpoints: Open Interest
// Statistics, Long/Short Ratio, Top Trader Long/Short Account and Position Ratios, and Taker Buy/Sell Volume
type USDTMarginedFuturesDataRequest struct {
	Symbol    currency.Pair
	Period    kline.Interval // 5m, 15m, 30m, 1h, 2h, 4h, 6h, 12h or 1d
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
}

// USDTMarginedFuturesLongShortRatioResponse holds a period's long/short ratio of accounts or positions
type USDTMarginedFuturesLongShortRatioResponse struct {
	Symbol         string       `json:"symbol"`
	LongShortRatio types.Number `json:"longShortRatio"`
	// LongAccount and ShortAccount are account shares on the account ratio endpoints and position shares on Top
	// Trader Long/Short Position Ratio, which keeps the same field names
	LongAccount  types.Number `json:"longAccount"`
	ShortAccount types.Number `json:"shortAccount"`
	Timestamp    types.Time   `json:"timestamp"`
}

// USDTMarginedFuturesMarkPriceResponse holds a symbol's mark price and funding rate
type USDTMarginedFuturesMarkPriceResponse struct {
	Symbol               string       `json:"symbol"`
	MarkPrice            types.Number `json:"markPrice"`
	IndexPrice           types.Number `json:"indexPrice"`
	EstimatedSettlePrice types.Number `json:"estimatedSettlePrice"`
	LastFundingRate      types.Number `json:"lastFundingRate"`
	InterestRate         types.Number `json:"interestRate"`
	NextFundingTime      types.Time   `json:"nextFundingTime"`
	Time                 types.Time   `json:"time"`
}

// AssetIndexResponse holds an asset's index price and Multi-Assets mode buffers and rates
type AssetIndexResponse struct {
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

// USDTMarginedFuturesPublicTradesResponse holds a market trade from Recent Trades List or Old Trades Lookup
type USDTMarginedFuturesPublicTradesResponse struct {
	ID                            uint64       `json:"id"`
	Price                         types.Number `json:"price"`
	Quantity                      types.Number `json:"qty"`
	QuoteQuantity                 types.Number `json:"quoteQty"`
	Time                          types.Time   `json:"time"`
	IsBuyerMaker                  bool         `json:"isBuyerMaker"`
	IsRetailPriceImprovementTrade bool         `json:"isRPITrade"`
}

// USDTMarginedFuturesOpenInterestResponse holds the reply of Open Interest
type USDTMarginedFuturesOpenInterestResponse struct {
	OpenInterest types.Number `json:"openInterest"`
	Symbol       string       `json:"symbol"`
	Time         types.Time   `json:"time"`
}

// USDTMarginedFuturesOpenInterestStatsResponse holds a period's open interest statistics
type USDTMarginedFuturesOpenInterestStatsResponse struct {
	Symbol               string       `json:"symbol"`
	SumOpenInterest      types.Number `json:"sumOpenInterest"`
	SumOpenInterestValue types.Number `json:"sumOpenInterestValue"`
	CMCCirculatingSupply types.Number `json:"CMCCirculatingSupply"` // circulating supply provided by CoinMarketCap
	Timestamp            types.Time   `json:"timestamp"`
}

// USDTMarginedFuturesOrderBookResponse holds the reply of Order Book and RPI Order Book
type USDTMarginedFuturesOrderBookResponse struct {
	LastUpdateID      uint64                           `json:"lastUpdateId"`
	MessageOutputTime types.Time                       `json:"E"`
	TransactionTime   types.Time                       `json:"T"`
	Bids              orderbook.LevelsArrayPriceAmount `json:"bids"`
	Asks              orderbook.LevelsArrayPriceAmount `json:"asks"`
}

// SettlementPriceResponse holds a quarterly contract's settlement price
type SettlementPriceResponse struct {
	DeliveryTime  types.Time `json:"deliveryTime"`
	DeliveryPrice float64    `json:"deliveryPrice"`
}

// USDTMarginedFuturesIndexPriceConstituentsResponse holds the reply of Query Index Price Constituents
type USDTMarginedFuturesIndexPriceConstituentsResponse struct {
	Symbol       string                                     `json:"symbol"`
	Time         types.Time                                 `json:"time"`
	Constituents []USDTMarginedFuturesIndexPriceConstituent `json:"constituents"`
}

// USDTMarginedFuturesIndexPriceConstituent holds one exchange's contribution to an index price; TradFi perps report a
// price of -1
type USDTMarginedFuturesIndexPriceConstituent struct {
	Exchange string       `json:"exchange"`
	Symbol   string       `json:"symbol"`
	Price    types.Number `json:"price"`
	Weight   types.Number `json:"weight"`
}

// USDTMarginedFuturesSymbolOrderbookTickerResponse holds a symbol's best bid and ask
type USDTMarginedFuturesSymbolOrderbookTickerResponse struct {
	Symbol       string       `json:"symbol"`
	BidPrice     types.Number `json:"bidPrice"`
	BidQuantity  types.Number `json:"bidQty"`
	AskPrice     types.Number `json:"askPrice"`
	AskQuantity  types.Number `json:"askQty"`
	Time         types.Time   `json:"time"`
	LastUpdateID uint64       `json:"lastUpdateId"` // not in the schema, but sent live
}

// USDTMarginedFuturesSymbolPriceTickerResponse holds a symbol's latest price
type USDTMarginedFuturesSymbolPriceTickerResponse struct {
	Symbol string       `json:"symbol"`
	Price  types.Number `json:"price"`
	Time   types.Time   `json:"time"`
}

// USDTMarginedFuturesTakerVolumeResponse holds a period's taker buy and sell volumes
type USDTMarginedFuturesTakerVolumeResponse struct {
	BuySellRatio types.Number `json:"buySellRatio"`
	BuyVolume    types.Number `json:"buyVol"`
	SellVolume   types.Number `json:"sellVol"`
	Timestamp    types.Time   `json:"timestamp"`
}

// USDTMarginedFutures24HourPriceChangeStatsResponse holds a symbol's 24 hour rolling window price change statistics
type USDTMarginedFutures24HourPriceChangeStatsResponse struct {
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

// USDTMarginedFuturesAccountTradeListRequest holds the parameters of Account Trade List
type USDTMarginedFuturesAccountTradeListRequest struct {
	Symbol    currency.Pair
	OrderID   uint64
	StartTime time.Time
	EndTime   time.Time
	FromID    uint64 // cannot be sent with StartTime or EndTime
	Limit     uint64
}

// USDTMarginedFuturesAccountTradeHistoryResponse holds one of the account's trades
type USDTMarginedFuturesAccountTradeHistoryResponse struct {
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
	RealisedPNL   types.Number  `json:"realizedPnl"`
	Side          Side          `json:"side"`
	PositionSide  PositionSide  `json:"positionSide"`
	Symbol        string        `json:"symbol"`
	Pair          string        `json:"pair"`
	Time          types.Time    `json:"time"`
}

// USDTMarginedFuturesAllOrdersRequest holds the parameters of All Orders
type USDTMarginedFuturesAllOrdersRequest struct {
	Symbol    currency.Pair
	OrderID   uint64 // returns orders with this ID and newer ones
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
}

// USDTMarginedFuturesOrderBase holds the fields every USDⓈ-M order reply carries
type USDTMarginedFuturesOrderBase struct {
	ClientOrderID           string                  `json:"clientOrderId"`
	ExecutedQuantity        types.Number            `json:"executedQty"`
	OrderID                 uint64                  `json:"orderId"`
	OriginalQuantity        types.Number            `json:"origQty"`
	Price                   types.Number            `json:"price"`
	ReduceOnly              bool                    `json:"reduceOnly"`
	Side                    Side                    `json:"side"`
	PositionSide            PositionSide            `json:"positionSide"`
	Status                  string                  `json:"status"`
	StopPrice               types.Number            `json:"stopPrice"` // ignore when the order type is TRAILING_STOP_MARKET
	ClosePosition           bool                    `json:"closePosition"`
	Symbol                  string                  `json:"symbol"`
	TimeInForce             TimeInForce             `json:"timeInForce"`
	Type                    OrderType               `json:"type"`
	OriginalType            OrderType               `json:"origType"`
	UpdateTime              types.Time              `json:"updateTime"`
	WorkingType             WorkingType             `json:"workingType"`
	PriceProtect            bool                    `json:"priceProtect"`
	PriceMatch              PriceMatch              `json:"priceMatch"`
	SelfTradePreventionMode SelfTradePreventionMode `json:"selfTradePreventionMode"`
	GoodTillDate            types.Time              `json:"goodTillDate"`
}

// USDTMarginedFuturesOrderResponse holds an order's state from Query Order, Query Current Open Order and Current All
// Open Orders
type USDTMarginedFuturesOrderResponse struct {
	USDTMarginedFuturesOrderBase
	AveragePrice    types.Number `json:"avgPrice"`
	CumulativeQuote types.Number `json:"cumQuote"`
	Time            types.Time   `json:"time"`
	ActivatePrice   types.Number `json:"activatePrice"` // only sent for TRAILING_STOP_MARKET orders
	PriceRate       types.Number `json:"priceRate"`     // callback rate, only sent for TRAILING_STOP_MARKET orders
}

// USDTMarginedFuturesAllOrdersResponse holds an order from All Orders, which may also list COIN-M orders
type USDTMarginedFuturesAllOrdersResponse struct {
	USDTMarginedFuturesOrderResponse
	CumulativeBase types.Number `json:"cumBase"`
	Pair           string       `json:"pair"`
}

// USDTMarginedFuturesAutoCancelAllOpenOrdersResponse holds the reply of Auto-Cancel All Open Orders
type USDTMarginedFuturesAutoCancelAllOpenOrdersResponse struct {
	Symbol                      string       `json:"symbol"`
	CountdownTimeInMilliseconds types.Number `json:"countdownTime"`
}

// USDTMarginedFuturesAlgoOrderBase holds the fields every USDⓈ-M algo (conditional) order reply carries
type USDTMarginedFuturesAlgoOrderBase struct {
	AlgoID                  uint64                  `json:"algoId"`
	ClientAlgoID            string                  `json:"clientAlgoId"`
	AlgoType                AlgoType                `json:"algoType"`
	OrderType               OrderType               `json:"orderType"`
	Symbol                  string                  `json:"symbol"`
	Side                    Side                    `json:"side"`
	PositionSide            PositionSide            `json:"positionSide"`
	TimeInForce             TimeInForce             `json:"timeInForce"`
	Quantity                types.Number            `json:"quantity"`
	AlgoStatus              string                  `json:"algoStatus"`
	TriggerPrice            types.Number            `json:"triggerPrice"`
	Price                   types.Number            `json:"price"`
	IcebergQuantity         string                  `json:"icebergQuantity"` // "null": algo orders take no iceberg quantity
	SelfTradePreventionMode SelfTradePreventionMode `json:"selfTradePreventionMode"`
	WorkingType             WorkingType             `json:"workingType"`
	PriceMatch              PriceMatch              `json:"priceMatch"`
	ClosePosition           bool                    `json:"closePosition"`
	PriceProtect            bool                    `json:"priceProtect"`
	ReduceOnly              bool                    `json:"reduceOnly"`
	CreateTime              types.Time              `json:"createTime"`
	UpdateTime              types.Time              `json:"updateTime"`
	TriggerTime             types.Time              `json:"triggerTime"`
	GoodTillDate            types.Time              `json:"goodTillDate"`
}

// USDTMarginedFuturesAlgoOrderResponse holds an algo order from Query Algo Order, Current All Algo Open Orders or Query
// All Algo Orders
type USDTMarginedFuturesAlgoOrderResponse struct {
	USDTMarginedFuturesAlgoOrderBase
	ActualOrderID          string       `json:"actualOrderId"` // empty until the order triggers
	ActualPrice            types.Number `json:"actualPrice"`   // the average fill price once filled
	ActualType             OrderType    `json:"actualType"`    // only sent once the order triggers
	ActualQuantity         types.Number `json:"actualQty"`     // only sent once the order fills
	TakeProfitTriggerPrice types.Number `json:"tpTriggerPrice"`
	TakeProfitPrice        types.Number `json:"tpPrice"`
	StopLossTriggerPrice   types.Number `json:"slTriggerPrice"`
	StopLossPrice          types.Number `json:"slPrice"`
	TakeProfitOrderType    OrderType    `json:"tpOrderType"`
}

// USDTMarginedFuturesNewAlgoOrderRequest holds the parameters of New Algo Order
type USDTMarginedFuturesNewAlgoOrderRequest struct {
	AlgoType     AlgoType // CONDITIONAL
	Symbol       currency.Pair
	Side         Side
	PositionSide PositionSide
	// OrderType is STOP_MARKET, TAKE_PROFIT_MARKET, STOP, TAKE_PROFIT or TRAILING_STOP_MARKET
	OrderType               OrderType
	TimeInForce             TimeInForce
	Quantity                float64 // cannot be sent with ClosePosition
	Price                   float64 // cannot be sent with PriceMatch
	TriggerPrice            float64
	WorkingType             WorkingType // MARK_PRICE or CONTRACT_PRICE
	PriceMatch              PriceMatch
	ClosePosition           bool
	PriceProtect            bool
	ReduceOnly              bool // cannot be sent with ClosePosition
	ActivatePrice           float64
	CallbackRate            float64
	ClientAlgoID            string
	NewOrderResponseType    NewOrderResponseType
	SelfTradePreventionMode SelfTradePreventionMode
	GoodTillDate            time.Time // required for GTD orders; Binance keeps second precision
}

// USDTMarginedFuturesNewAlgoOrderResponse holds the reply of New Algo Order
type USDTMarginedFuturesNewAlgoOrderResponse struct {
	USDTMarginedFuturesAlgoOrderBase
	ActivatePrice types.Number `json:"activatePrice"` // only set for TRAILING_STOP_MARKET orders
	CallbackRate  types.Number `json:"callbackRate"`  // only set for TRAILING_STOP_MARKET orders
}

// USDTMarginedFuturesCancelAlgoOrderResponse holds the reply of Cancel Algo Order
type USDTMarginedFuturesCancelAlgoOrderResponse struct {
	AlgoID       uint64       `json:"algoId"`
	ClientAlgoID string       `json:"clientAlgoId"`
	Code         types.Number `json:"code"`
	Message      string       `json:"msg"`
}

// USDTMarginedFuturesModifyOrderRequest holds the parameters of Modify Order and one order of Modify Multiple Orders
type USDTMarginedFuturesModifyOrderRequest struct {
	OrderID           uint64 // OrderID wins when both IDs are set
	OrigClientOrderID string
	Symbol            currency.Pair
	Side              Side
	Quantity          float64
	Price             float64 // required unless PriceMatch is set, and cannot be sent with it
	PriceMatch        PriceMatch
	ModifyID          uint64 // returned unchanged in the reply
	// ReduceOnly skips the minimum notional check when the original order is reduce-only, and is rejected otherwise
	ReduceOnly bool
}

// USDTMarginedFuturesNewOrderResponse holds the reply of New Order
type USDTMarginedFuturesNewOrderResponse struct {
	USDTMarginedFuturesOrderBase
	CumulativeQuantity types.Number `json:"cumQty"`
}

// USDTMarginedFuturesModifyOrderResponse holds the reply of Modify Order
type USDTMarginedFuturesModifyOrderResponse struct {
	USDTMarginedFuturesNewOrderResponse
	Pair     string `json:"pair"`
	ModifyID uint64 `json:"modifyId"` // only sent when the request set it
}

// USDTMarginedFuturesModifyBatchOrderResponse holds one order's reply from Modify Multiple Orders: the modified order,
// or the code and message that rejected it
type USDTMarginedFuturesModifyBatchOrderResponse struct {
	USDTMarginedFuturesModifyOrderResponse
	Code    int64  `json:"code"` // negative when Binance rejected the entry
	Message string `json:"msg"`
}

// USDTMarginedFuturesNewOrderRequest holds the parameters of New Order and one order of Place Multiple Orders. Stop,
// take profit and trailing stop orders are placed with USDTMarginedFuturesNewAlgoOrder
type USDTMarginedFuturesNewOrderRequest struct {
	Symbol                  currency.Pair
	Side                    Side
	PositionSide            PositionSide
	OrderType               OrderType // LIMIT or MARKET
	TimeInForce             TimeInForce
	ReduceOnly              bool
	Quantity                float64
	Price                   float64 // cannot be sent with PriceMatch
	NewClientOrderID        string
	NewOrderResponseType    NewOrderResponseType
	PriceMatch              PriceMatch
	SelfTradePreventionMode SelfTradePreventionMode
	GoodTillDate            time.Time // required for GTD orders; Binance keeps second precision
}

// USDTMarginedFuturesPlaceBatchOrderResponse holds one order's reply from Place Multiple Orders: the new order, or the
// code and message that rejected it
type USDTMarginedFuturesPlaceBatchOrderResponse struct {
	USDTMarginedFuturesNewOrderResponse
	Code    int64  `json:"code"` // negative when Binance rejected the entry
	Message string `json:"msg"`
}

// USDTMarginedFuturesCancelOrderResponse holds the reply of Cancel Order
type USDTMarginedFuturesCancelOrderResponse struct {
	USDTMarginedFuturesNewOrderResponse
	ActivatePrice types.Number `json:"activatePrice"` // only sent for TRAILING_STOP_MARKET orders
	PriceRate     types.Number `json:"priceRate"`     // callback rate, only sent for TRAILING_STOP_MARKET orders
}

// USDTMarginedFuturesCancelBatchOrderResponse holds one order's reply from Cancel Multiple Orders: the cancelled order,
// or the code and message that rejected it
type USDTMarginedFuturesCancelBatchOrderResponse struct {
	USDTMarginedFuturesCancelOrderResponse
	Code    int64  `json:"code"` // negative when Binance rejected the entry
	Message string `json:"msg"`
}

// USDTMarginedFuturesChangeInitialLeverageResponse holds the reply of Change Initial Leverage
type USDTMarginedFuturesChangeInitialLeverageResponse struct {
	Leverage         uint64       `json:"leverage"`
	MaxNotionalValue types.Number `json:"maxNotionalValue"`
	Symbol           string       `json:"symbol"`
}

// USDTMarginedFuturesOrderModifyHistoryRequest holds the parameters of Get Order Modify History
type USDTMarginedFuturesOrderModifyHistoryRequest struct {
	Symbol            currency.Pair
	OrderID           uint64 // OrderID wins when both IDs are set
	OrigClientOrderID string
	StartTime         time.Time
	EndTime           time.Time
	Limit             uint64
}

// FuturesOrderAmendmentResponse holds one modification from Get Order Modify History
type FuturesOrderAmendmentResponse struct {
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

// USDTMarginedFuturesPositionMarginChangeHistoryRequest holds the parameters of Get Position Margin Change History
type USDTMarginedFuturesPositionMarginChangeHistoryRequest struct {
	Symbol currency.Pair
	// Type filters by direction: 1 adds position margin and 2 reduces it; both when zero
	Type      uint64
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
}

// USDTMarginedFuturesPositionMarginChangeResponse holds one isolated position margin change
type USDTMarginedFuturesPositionMarginChangeResponse struct {
	Symbol       string        `json:"symbol"`
	Type         uint64        `json:"type"` // 1 adds and 2 reduces position margin
	DeltaType    string        `json:"deltaType"`
	Amount       types.Number  `json:"amount"`
	Asset        currency.Code `json:"asset"`
	Time         types.Time    `json:"time"`
	PositionSide PositionSide  `json:"positionSide"`
}

// USDTMarginedFuturesModifyIsolatedPositionMarginResponse holds the reply of Modify Isolated Position Margin
type USDTMarginedFuturesModifyIsolatedPositionMarginResponse struct {
	Amount  float64 `json:"amount"`
	Code    int64   `json:"code"` // 200 on success; Binance's error codes are negative
	Message string  `json:"msg"`
	Type    uint64  `json:"type"` // 1 adds and 2 reduces position margin
}

// USDTMarginedFuturesPositionAutoDeleveragingQuantileResponse holds a symbol's auto-deleveraging queue estimates
type USDTMarginedFuturesPositionAutoDeleveragingQuantileResponse struct {
	Symbol                   string                                      `json:"symbol"`
	AutoDeleveragingQuantile USDTMarginedFuturesAutoDeleveragingQuantile `json:"adlQuantile"`
}

// USDTMarginedFuturesAutoDeleveragingQuantile holds the auto-deleveraging quantile, 0 (lowest) to 4 (highest), of each
// position side
type USDTMarginedFuturesAutoDeleveragingQuantile struct {
	Long  uint64 `json:"LONG"`
	Short uint64 `json:"SHORT"`
	// Hedge is only a sign that cross positions in Hedge Mode share one quantile, sent instead of Both; ignore its value
	Hedge uint64 `json:"HEDGE"`
	Both  uint64 `json:"BOTH"`
}

// USDTMarginedFuturesPositionInformationV2Response holds a position from Position Information V2
type USDTMarginedFuturesPositionInformationV2Response struct {
	Symbol           string        `json:"symbol"`
	PositionAmount   types.Number  `json:"positionAmt"`
	EntryPrice       types.Number  `json:"entryPrice"`
	BreakEvenPrice   types.Number  `json:"breakEvenPrice"`
	MarkPrice        types.Number  `json:"markPrice"`
	UnrealisedProfit types.Number  `json:"unRealizedProfit"`
	LiquidationPrice types.Number  `json:"liquidationPrice"`
	Leverage         types.Number  `json:"leverage"`
	MaxNotionalValue types.Number  `json:"maxNotionalValue"`
	MarginType       string        `json:"marginType"`
	IsolatedMargin   types.Number  `json:"isolatedMargin"`
	IsAutoAddMargin  types.Boolean `json:"isAutoAddMargin"`
	PositionSide     PositionSide  `json:"positionSide"`
	Notional         types.Number  `json:"notional"`
	IsolatedWallet   types.Number  `json:"isolatedWallet"`
	UpdateTime       types.Time    `json:"updateTime"`
}

// USDTMarginedFuturesPositionInformationV3Response holds a position from Position Information V3, which only lists
// symbols with a position or open orders and leaves leverage and margin type to Symbol Configuration
type USDTMarginedFuturesPositionInformationV3Response struct {
	Symbol                  string        `json:"symbol"`
	PositionSide            PositionSide  `json:"positionSide"`
	PositionAmount          types.Number  `json:"positionAmt"`
	EntryPrice              types.Number  `json:"entryPrice"`
	BreakEvenPrice          types.Number  `json:"breakEvenPrice"`
	MarkPrice               types.Number  `json:"markPrice"`
	UnrealisedProfit        types.Number  `json:"unRealizedProfit"`
	LiquidationPrice        types.Number  `json:"liquidationPrice"`
	IsolatedMargin          types.Number  `json:"isolatedMargin"`
	Notional                types.Number  `json:"notional"`
	MarginAsset             currency.Code `json:"marginAsset"`
	IsolatedWallet          types.Number  `json:"isolatedWallet"`
	InitialMargin           types.Number  `json:"initialMargin"`
	MaintenanceMargin       types.Number  `json:"maintMargin"`
	PositionInitialMargin   types.Number  `json:"positionInitialMargin"`
	OpenOrderInitialMargin  types.Number  `json:"openOrderInitialMargin"`
	AutoDeleveragingRanking uint64        `json:"adl"`
	BidNotional             types.Number  `json:"bidNotional"`
	AskNotional             types.Number  `json:"askNotional"`
	UpdateTime              types.Time    `json:"updateTime"`
}

// USDTMarginedFuturesAllAlgoOrdersRequest holds the parameters of Query All Algo Orders
type USDTMarginedFuturesAllAlgoOrdersRequest struct {
	Symbol    currency.Pair
	AlgoID    uint64 // returns algo orders with this ID and newer ones
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
}

// USDTMarginedFuturesForceOrdersRequest holds the parameters of User's Force Orders
type USDTMarginedFuturesForceOrdersRequest struct {
	Symbol        currency.Pair
	AutoCloseType AutoCloseType // LIQUIDATION or ADL; empty for both
	StartTime     time.Time
	EndTime       time.Time
	Limit         uint64
}

// USDTMarginedFuturesForceOrderResponse holds a liquidation or auto-deleveraging order from User's Force Orders, which
// may also list COIN-M orders
type USDTMarginedFuturesForceOrderResponse struct {
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
	TimeInForce      TimeInForce  `json:"timeInForce"`
	Type             OrderType    `json:"type"`
	ReduceOnly       bool         `json:"reduceOnly"`
	ClosePosition    bool         `json:"closePosition"`
	Side             Side         `json:"side"`
	PositionSide     PositionSide `json:"positionSide"`
	StopPrice        types.Number `json:"stopPrice"`
	WorkingType      WorkingType  `json:"workingType"`
	OriginalType     OrderType    `json:"origType"`
	Time             types.Time   `json:"time"`
	UpdateTime       types.Time   `json:"updateTime"`
}

// FuturesListenKeyResponse holds the listen key of a USDⓈ-M or COIN-M user data stream
type FuturesListenKeyResponse struct {
	ListenKey string `json:"listenKey"`
}
