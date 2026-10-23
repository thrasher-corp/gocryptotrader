package bybit

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// ServerTimeResponse holds Bybit's server time
type ServerTimeResponse struct {
	TimeSecond types.Time `json:"timeSecond"`
	TimeNano   types.Time `json:"timeNano"`
}

// KlineRequest holds the parameters for GetKlines, GetMarkPriceKline, GetIndexPriceKline and GetPremiumIndexPriceKline
type KlineRequest struct {
	// Category defaults to linear when empty
	Category string
	// Symbol is required
	Symbol string
	// Interval is required
	Interval kline.Interval
	Start    time.Time
	End      time.Time
	Limit    uint64
}

// KlineResponse holds a symbol's candles, newest first
type KlineResponse struct {
	Category string      `json:"category"`
	Symbol   string      `json:"symbol"`
	List     []KlineItem `json:"list"`
}

// KlineItem holds a candle, which Bybit sends as an array of strings
type KlineItem struct {
	StartTime types.Time
	OpenPrice types.Number
	HighPrice types.Number
	LowPrice  types.Number
	// ClosePrice is the last traded price while the candle is open
	ClosePrice types.Number
	// Volume is in the base coin, except for inverse contracts, where it is in the quote coin
	Volume types.Number
	// Turnover is in the quote coin, except for inverse contracts, where it is in the base coin
	Turnover types.Number
}

// UnmarshalJSON decodes a candle from its array form
func (k *KlineItem) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[7]any{&k.StartTime, &k.OpenPrice, &k.HighPrice, &k.LowPrice, &k.ClosePrice, &k.Volume, &k.Turnover})
}

// PriceKlineResponse holds a symbol's mark, index or premium index price candles, newest first
type PriceKlineResponse struct {
	Category string           `json:"category"`
	Symbol   string           `json:"symbol"`
	List     []PriceKlineItem `json:"list"`
}

// PriceKlineItem holds a price candle, which has no volume or turnover and which Bybit sends as an array of strings
type PriceKlineItem struct {
	StartTime types.Time
	OpenPrice types.Number
	HighPrice types.Number
	LowPrice  types.Number
	// ClosePrice is the last price while the candle is open
	ClosePrice types.Number
}

// UnmarshalJSON decodes a price candle from its array form
func (k *PriceKlineItem) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[5]any{&k.StartTime, &k.OpenPrice, &k.HighPrice, &k.LowPrice, &k.ClosePrice})
}

// InstrumentInfoRequest holds the parameters for GetInstrumentInfo
type InstrumentInfoRequest struct {
	// Category is required
	Category   string
	Symbol     string
	SymbolType string
	Status     string
	// BaseCoin applies to linear, inverse and option; option takes "All" for every base coin, which a currency.Code
	// cannot carry since Bybit matches it case sensitively
	BaseCoin string
	Limit    uint64
	Cursor   string
}

// InstrumentsInfoResponse holds a page of instrument specifications
type InstrumentsInfoResponse struct {
	Category       string           `json:"category"`
	List           []InstrumentInfo `json:"list"`
	NextPageCursor string           `json:"nextPageCursor"`
}

// InstrumentInfo holds an instrument's specification. Each category returns a subset of the fields
type InstrumentInfo struct {
	Symbol       string        `json:"symbol"`
	SymbolID     uint64        `json:"symbolId"`
	ContractType string        `json:"contractType"`
	OptionsType  string        `json:"optionsType"`
	Status       string        `json:"status"`
	BaseCoin     currency.Code `json:"baseCoin"`
	QuoteCoin    currency.Code `json:"quoteCoin"`
	SettleCoin   currency.Code `json:"settleCoin"`
	SymbolType   string        `json:"symbolType"`
	// Innovation is deprecated in favour of SymbolType
	Innovation types.Boolean `json:"innovation"`
	// XStockMultiplier relates an xStocks token to its stock: stock price = token price / multiplier
	XStockMultiplier    types.Number  `json:"xstockMultiplier"`
	MarginTrading       string        `json:"marginTrading"`
	SpecialTreatmentTag types.Boolean `json:"stTag"`
	LaunchTime          types.Time    `json:"launchTime"`
	// DeliveryTime is the expiry of dated contracts and options, and the delisting time of perpetual contracts
	DeliveryTime       types.Time     `json:"deliveryTime"`
	DeliveryFeeRate    types.Number   `json:"deliveryFeeRate"`
	PriceScale         types.Number   `json:"priceScale"`
	LeverageFilter     LeverageFilter `json:"leverageFilter"`
	PriceFilter        PriceFilter    `json:"priceFilter"`
	LotSizeFilter      LotSizeFilter  `json:"lotSizeFilter"`
	UnifiedMarginTrade bool           `json:"unifiedMarginTrade"`
	// FundingInterval is in minutes
	FundingInterval  uint64       `json:"fundingInterval"`
	CopyTrading      string       `json:"copyTrading"`
	UpperFundingRate types.Number `json:"upperFundingRate"`
	LowerFundingRate types.Number `json:"lowerFundingRate"`
	DisplayName      string       `json:"displayName"`
	FullName         string       `json:"fullName"`
	// MarketRegion and UnderlyingTicker describe the underlying asset of TradFi perpetual contracts
	MarketRegion                  string         `json:"marketRegion"`
	UnderlyingTicker              string         `json:"underlyingTicker"`
	Tags                          []string       `json:"tags"`
	ForbidUnrealisedPNLWithdrawal bool           `json:"forbidUplWithdrawal"`
	RiskParameters                RiskParameters `json:"riskParameters"`
	IsPreListing                  bool           `json:"isPreListing"`
	// PreListingInfo is nil unless IsPreListing is true
	PreListingInfo *PreListingInfo `json:"preListingInfo"`
}

// LeverageFilter holds an instrument's leverage limits
type LeverageFilter struct {
	MinLeverage  types.Number `json:"minLeverage"`
	MaxLeverage  types.Number `json:"maxLeverage"`
	LeverageStep types.Number `json:"leverageStep"`
}

// PriceFilter holds an instrument's order price limits
type PriceFilter struct {
	MinPrice types.Number `json:"minPrice"`
	MaxPrice types.Number `json:"maxPrice"`
	TickSize types.Number `json:"tickSize"`
}

// LotSizeFilter holds an instrument's order size limits. Each category returns a subset of the fields
type LotSizeFilter struct {
	MaxOrderQuantity types.Number `json:"maxOrderQty"`
	MinOrderQuantity types.Number `json:"minOrderQty"`
	QuantityStep     types.Number `json:"qtyStep"`
	// PostOnlyMaxOrderQuantity is deprecated in favour of MaxOrderQuantity
	PostOnlyMaxOrderQuantity types.Number `json:"postOnlyMaxOrderQty"`
	// MaxMktOrderQuantity is the linear and inverse market order limit; it keeps the vendor's abbreviation because
	// spot's own market order limit, maxMarketOrderQty, expands to the same name
	MaxMktOrderQuantity       types.Number `json:"maxMktOrderQty"`
	MinNotionalValue          types.Number `json:"minNotionalValue"`
	BasePrecision             types.Number `json:"basePrecision"`
	QuotePrecision            types.Number `json:"quotePrecision"`
	MinOrderAmount            types.Number `json:"minOrderAmt"`
	MaxOrderAmount            types.Number `json:"maxOrderAmt"`
	MaxLimitOrderQuantity     types.Number `json:"maxLimitOrderQty"`
	MaxMarketOrderQuantity    types.Number `json:"maxMarketOrderQty"`
	PostOnlyMaxLimitOrderSize types.Number `json:"postOnlyMaxLimitOrderSize"`
}

// RiskParameters holds the ratios that limit an instrument's limit order prices
type RiskParameters struct {
	PriceLimitRatioX types.Number `json:"priceLimitRatioX"`
	PriceLimitRatioY types.Number `json:"priceLimitRatioY"`
}

// PreListingInfo holds a pre-market contract's auction schedule and fees
type PreListingInfo struct {
	CurrentAuctionPhase string            `json:"curAuctionPhase"`
	Phases              []PreListingPhase `json:"phases"`
	AuctionFeeInfo      AuctionFeeInfo    `json:"auctionFeeInfo"`
	SkipCallAuction     bool              `json:"skipCallAuction"`
}

// PreListingPhase holds a pre-market trading phase and its time window
type PreListingPhase struct {
	Phase     string              `json:"phase"`
	StartTime PreListingPhaseTime `json:"startTime"`
	EndTime   PreListingPhaseTime `json:"endTime"`
}

// preListingSkippedPhaseTime is the time Bybit sends, in Unix milliseconds, for the phases a pre-market contract
// skips. It is 2000-01-01, which types.Time would read as a 13 digit timestamp missing a digit, in the year 2269
const preListingSkippedPhaseTime = "946684800000"

// PreListingPhaseTime is a pre-market phase's start or end time, which Bybit sends in Unix milliseconds. The times of
// the phases a contract skips, and the end of continuous trading, which has none, decode as the zero time
type PreListingPhaseTime time.Time

// UnmarshalJSON decodes a quoted or bare Unix millisecond timestamp
func (p *PreListingPhaseTime) UnmarshalJSON(data []byte) error {
	switch value := strings.Trim(string(data), `"`); value {
	case "", "0", "null", preListingSkippedPhaseTime:
		*p = PreListingPhaseTime{}
	default:
		milliseconds, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("%w: %s: %w", types.ErrInvalidTimestampFormat, data, err)
		}
		*p = PreListingPhaseTime(time.UnixMilli(milliseconds))
	}
	return nil
}

// MarshalJSON encodes the time in RFC 3339 format, as types.Time does
func (p PreListingPhaseTime) MarshalJSON() ([]byte, error) {
	return time.Time(p).MarshalJSON()
}

// Time returns the phase time, the zero time when the phase has none
func (p PreListingPhaseTime) Time() time.Time {
	return time.Time(p)
}

// AuctionFeeInfo holds a pre-market contract's fee rates
type AuctionFeeInfo struct {
	// AuctionFeeRate applies during the auction phases; no fee is charged until continuous trading starts
	AuctionFeeRate types.Number `json:"auctionFeeRate"`
	TakerFeeRate   types.Number `json:"takerFeeRate"`
	MakerFeeRate   types.Number `json:"makerFeeRate"`
}

// OrderbookResponse holds an orderbook snapshot, which excludes retail price improvement orders
type OrderbookResponse struct {
	Symbol string                           `json:"s"`
	Bids   orderbook.LevelsArrayPriceAmount `json:"b"`
	Asks   orderbook.LevelsArrayPriceAmount `json:"a"`
	// Timestamp is when the system generated the data
	Timestamp types.Time `json:"ts"`
	// UpdateID matches the u of the websocket orderbook stream at the same depth
	UpdateID uint64 `json:"u"`
	// Sequence is the cross sequence, which orders data across orderbook depths
	Sequence uint64 `json:"seq"`
	// CreationTimestamp is when the matching engine produced the data; it correlates with the T of public trades
	CreationTimestamp types.Time `json:"cts"`
}

// RPIOrderbookResponse holds an orderbook snapshot that separates retail price improvement (RPI) order sizes
type RPIOrderbookResponse struct {
	Symbol            string              `json:"s"`
	Bids              []RPIOrderbookLevel `json:"b"`
	Asks              []RPIOrderbookLevel `json:"a"`
	Timestamp         types.Time          `json:"ts"`
	UpdateID          uint64              `json:"u"`
	Sequence          uint64              `json:"seq"`
	CreationTimestamp types.Time          `json:"cts"`
}

// RPIOrderbookLevel is a price level that splits its size between ordinary and Retail Price Improvement orders. Bybit
// sends it as a [price, size, RPI size] array, over REST and the RPI orderbook stream alike
type RPIOrderbookLevel struct {
	Price types.Number
	// NonRPISize is the size of the orders that are not RPI orders; 0 in a delta, with an RPISize of 0, removes the
	// level
	NonRPISize types.Number
	// RPISize is hidden, as 0, while an RPI order crosses an ordinary order on the other side
	RPISize types.Number
}

// UnmarshalJSON decodes a [price, size, RPI size] level
func (l *RPIOrderbookLevel) UnmarshalJSON(data []byte) error {
	var level []types.Number
	if err := json.Unmarshal(data, &level); err != nil {
		return err
	}
	if len(level) != 3 {
		return fmt.Errorf("%w: %s", errInvalidRPIOrderbookLevel, data)
	}
	l.Price, l.NonRPISize, l.RPISize = level[0], level[1], level[2]
	return nil
}

// TickersRequest holds the parameters for GetTickers
type TickersRequest struct {
	// Category is required
	Category string
	Symbol   string
	// BaseCoin and ExpiryDate apply to option only, where either Symbol or BaseCoin is required
	BaseCoin   currency.Code
	ExpiryDate time.Time
}

// TickersResponse holds tickers
type TickersResponse struct {
	Category string   `json:"category"`
	List     []Ticker `json:"list"`
}

// Ticker holds a symbol's latest prices and 24 hour statistics. Linear and inverse, option and spot tickers each
// return a subset of the fields
type Ticker struct {
	Symbol                  string       `json:"symbol"`
	LastPrice               types.Number `json:"lastPrice"`
	IndexPrice              types.Number `json:"indexPrice"`
	MarkPrice               types.Number `json:"markPrice"`
	PreviousPrice24Hour     types.Number `json:"prevPrice24h"`
	Price24HourPercent      types.Number `json:"price24hPcnt"`
	HighPrice24Hour         types.Number `json:"highPrice24h"`
	LowPrice24Hour          types.Number `json:"lowPrice24h"`
	PreviousPrice1Hour      types.Number `json:"prevPrice1h"`
	OpenInterest            types.Number `json:"openInterest"`
	OpenInterestValue       types.Number `json:"openInterestValue"`
	SingleOpenInterest      types.Number `json:"singleOpenInterest"`
	SingleOpenInterestValue types.Number `json:"singleOpenInterestValue"`
	// Turnover24Hour is in the quote coin and Volume24Hour in the base coin, except for inverse contracts, where the
	// two swap
	Turnover24Hour  types.Number `json:"turnover24h"`
	Volume24Hour    types.Number `json:"volume24h"`
	FundingRate     types.Number `json:"fundingRate"`
	NextFundingTime types.Time   `json:"nextFundingTime"`
	// PredictedDeliveryPrice has a value only in the 30 minutes before delivery
	PredictedDeliveryPrice types.Number `json:"predictedDeliveryPrice"`
	BasisRate              types.Number `json:"basisRate"`
	Basis                  types.Number `json:"basis"`
	DeliveryFeeRate        types.Number `json:"deliveryFeeRate"`
	DeliveryTime           types.Time   `json:"deliveryTime"`
	Bid1Price              types.Number `json:"bid1Price"`
	Bid1Size               types.Number `json:"bid1Size"`
	Bid1ImpliedVolatility  types.Number `json:"bid1Iv"`
	Ask1Price              types.Number `json:"ask1Price"`
	Ask1Size               types.Number `json:"ask1Size"`
	Ask1ImpliedVolatility  types.Number `json:"ask1Iv"`
	// PreOpenPrice and PreQuantity estimate a pre-market contract's opening and are meaningless once it opens
	PreOpenPrice           types.Number `json:"preOpenPrice"`
	PreQuantity            types.Number `json:"preQty"`
	CurrentPreListingPhase string       `json:"curPreListingPhase"`
	FundingIntervalHour    types.Number `json:"fundingIntervalHour"`
	FundingCap             types.Number `json:"fundingCap"`
	BasisRateYear          types.Number `json:"basisRateYear"`
	MarkImpliedVolatility  types.Number `json:"markIv"`
	UnderlyingPrice        types.Number `json:"underlyingPrice"`
	TotalVolume            types.Number `json:"totalVolume"`
	TotalTurnover          types.Number `json:"totalTurnover"`
	Delta                  types.Number `json:"delta"`
	Gamma                  types.Number `json:"gamma"`
	Vega                   types.Number `json:"vega"`
	Theta                  types.Number `json:"theta"`
	Change24Hour           types.Number `json:"change24h"`
	// USDIndexPrice values a spot coin in USD for the unified account; it is empty for non-collateral coins
	USDIndexPrice types.Number `json:"usdIndexPrice"`
}

// FundingRateHistoryRequest holds the parameters for GetFundingRateHistory
type FundingRateHistoryRequest struct {
	// Category and Symbol are required
	Category string
	Symbol   string
	// StartTime needs EndTime; EndTime alone returns the records up to it
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
}

// FundingRateHistoryResponse holds a symbol's funding rates, newest first
type FundingRateHistoryResponse struct {
	Category string        `json:"category"`
	List     []FundingRate `json:"list"`
}

// FundingRate holds a settled funding rate
type FundingRate struct {
	Symbol               string       `json:"symbol"`
	FundingRate          types.Number `json:"fundingRate"`
	FundingRateTimestamp types.Time   `json:"fundingRateTimestamp"`
}

// PublicTradingHistoryRequest holds the parameters for GetPublicTradingHistory
type PublicTradingHistoryRequest struct {
	// Category is required
	Category string
	// Symbol is required for spot, linear and inverse
	Symbol string
	// BaseCoin and OptionType apply to option only; BaseCoin defaults to BTC
	BaseCoin   currency.Code
	OptionType string
	Limit      uint64
}

// PublicTradingHistoryResponse holds recent public trades
type PublicTradingHistoryResponse struct {
	Category string        `json:"category"`
	List     []PublicTrade `json:"list"`
}

// PublicTrade holds a public trade. MarkPrice, IndexPrice, MarkImpliedVolatility and ImpliedVolatility apply to
// option only
type PublicTrade struct {
	ExecutionID string       `json:"execId"`
	Symbol      string       `json:"symbol"`
	Price       types.Number `json:"price"`
	Size        types.Number `json:"size"`
	// Side is the taker's side
	Side                  string       `json:"side"`
	Time                  types.Time   `json:"time"`
	IsBlockTrade          bool         `json:"isBlockTrade"`
	IsRPITrade            bool         `json:"isRPITrade"`
	MarkPrice             types.Number `json:"mP"`
	IndexPrice            types.Number `json:"iP"`
	MarkImpliedVolatility types.Number `json:"mIv"`
	ImpliedVolatility     types.Number `json:"iv"`
	// Sequence is the cross sequence
	Sequence types.Number `json:"seq"`
}

// OpenInterestRequest holds the parameters for GetOpenInterestData
type OpenInterestRequest struct {
	// Category, Symbol and IntervalTime (5min, 15min, 30min, 1h, 4h or 1d) are required
	Category     string
	Symbol       string
	IntervalTime string
	StartTime    time.Time
	EndTime      time.Time
	Limit        uint64
	Cursor       string
}

// OpenInterestResponse holds a symbol's open interest, newest first
type OpenInterestResponse struct {
	Category       string         `json:"category"`
	Symbol         string         `json:"symbol"`
	List           []OpenInterest `json:"list"`
	NextPageCursor string         `json:"nextPageCursor"`
}

// OpenInterest holds a symbol's open interest at a time, in the base coin for linear contracts and the quote coin
// for inverse contracts
type OpenInterest struct {
	// OpenInterest sums both sides
	OpenInterest types.Number `json:"openInterest"`
	// SingleOpenInterest counts one side
	SingleOpenInterest types.Number `json:"singleOpenInterest"`
	Timestamp          types.Time   `json:"timestamp"`
}

// HistoricalVolatilityRequest holds the parameters for GetHistoricalVolatility
type HistoricalVolatilityRequest struct {
	// Category is required
	Category string
	// BaseCoin defaults to BTC and QuoteCoin, USD or USDT, to USD, whose series ends on 26 December 2025
	BaseCoin  currency.Code
	QuoteCoin currency.Code
	// Period is the averaging period in days, 7 by default
	Period uint64
	// StartTime and EndTime go together; without them the latest hour is returned
	StartTime time.Time
	EndTime   time.Time
}

// HistoricalVolatilityResponse holds an option base coin's hourly historical volatility
type HistoricalVolatilityResponse struct {
	Period uint64       `json:"period"`
	Value  types.Number `json:"value"`
	Time   types.Time   `json:"time"`
}

// InsuranceResponse holds insurance pool balances
type InsuranceResponse struct {
	UpdatedTime types.Time      `json:"updatedTime"`
	List        []InsurancePool `json:"list"`
}

// InsurancePool holds an insurance pool's balance
type InsurancePool struct {
	Coin currency.Code `json:"coin"`
	// Symbols lists the contracts sharing the pool, comma separated; an isolated pool lists one
	Symbols string       `json:"symbols"`
	Balance types.Number `json:"balance"`
	// Value is in USD
	Value types.Number `json:"value"`
}

// RiskLimitTiersResponse holds a page of risk limit tiers
type RiskLimitTiersResponse struct {
	Category       string          `json:"category"`
	List           []RiskLimitTier `json:"list"`
	NextPageCursor string          `json:"nextPageCursor"`
}

// RiskLimitTier holds a risk limit tier's margin parameters
type RiskLimitTier struct {
	ID     uint64 `json:"id"`
	Symbol string `json:"symbol"`
	// RiskLimitValue is the tier's position limit
	RiskLimitValue types.Number `json:"riskLimitValue"`
	// MaintenanceMargin and InitialMargin are rates, documented as numbers but sent as strings
	MaintenanceMargin          types.Number  `json:"maintenanceMargin"`
	InitialMargin              types.Number  `json:"initialMargin"`
	IsLowestRisk               types.Boolean `json:"isLowestRisk"`
	MaxLeverage                types.Number  `json:"maxLeverage"`
	MaintenanceMarginDeduction types.Number  `json:"mmDeduction"`
}

// DeliveryPriceRequest holds the parameters for GetDeliveryPrice
type DeliveryPriceRequest struct {
	// Category is required
	Category string
	Symbol   string
	// BaseCoin applies to option only and defaults to BTC; SettleCoin defaults to USDC
	BaseCoin   currency.Code
	SettleCoin currency.Code
	Limit      uint64
	Cursor     string
}

// DeliveryPriceResponse holds a page of delivery prices
type DeliveryPriceResponse struct {
	Category       string          `json:"category"`
	List           []DeliveryPrice `json:"list"`
	NextPageCursor string          `json:"nextPageCursor"`
}

// DeliveryPrice holds a symbol's delivery price
type DeliveryPrice struct {
	Symbol        string       `json:"symbol"`
	DeliveryPrice types.Number `json:"deliveryPrice"`
	DeliveryTime  types.Time   `json:"deliveryTime"`
}

// NewDeliveryPriceResponse holds an option base coin's historical delivery prices, newest first
type NewDeliveryPriceResponse struct {
	Category string             `json:"category"`
	List     []NewDeliveryPrice `json:"list"`
}

// NewDeliveryPrice holds an option delivery price
type NewDeliveryPrice struct {
	DeliveryPrice types.Number `json:"deliveryPrice"`
	DeliveryTime  types.Time   `json:"deliveryTime"`
}

// LongShortRatioRequest holds the parameters for GetLongShortRatio
type LongShortRatioRequest struct {
	// Category, Symbol and Period (5min, 15min, 30min, 1h, 4h or 1d) are required
	Category  string
	Symbol    string
	Period    string
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
	Cursor    string
}

// LongShortRatioResponse holds a page of long short account ratios, newest first
type LongShortRatioResponse struct {
	List           []LongShortRatio `json:"list"`
	NextPageCursor string           `json:"nextPageCursor"`
}

// LongShortRatio holds the shares of position holders that are long and short
type LongShortRatio struct {
	Symbol    string       `json:"symbol"`
	BuyRatio  types.Number `json:"buyRatio"`
	SellRatio types.Number `json:"sellRatio"`
	Timestamp types.Time   `json:"timestamp"`
}

// IndexPriceComponentsResponse holds the exchange prices that make up an index price
type IndexPriceComponentsResponse struct {
	IndexName  string                `json:"indexName"`
	LastPrice  types.Number          `json:"lastPrice"`
	UpdateTime types.Time            `json:"updateTime"`
	Components []IndexPriceComponent `json:"components"`
}

// IndexPriceComponent holds an exchange's contribution to an index price
type IndexPriceComponent struct {
	Exchange        string       `json:"exchange"`
	SpotPair        string       `json:"spotPair"`
	EquivalentPrice types.Number `json:"equivalentPrice"`
	Multiplier      types.Number `json:"multiplier"`
	Price           types.Number `json:"price"`
	Weight          types.Number `json:"weight"`
}

// OrderPriceLimitResponse holds a symbol's order price limits
type OrderPriceLimitResponse struct {
	Symbol string `json:"symbol"`
	// BuyLimit is the highest bid price and SellLimit the lowest ask price
	BuyLimit  types.Number `json:"buyLmt"`
	SellLimit types.Number `json:"sellLmt"`
	Timestamp types.Time   `json:"ts"`
}

// ADLAlertResponse holds auto-deleveraging alerts and insurance pool data
type ADLAlertResponse struct {
	// UpdatedTime is documented as updateTime but sent as updatedTime
	UpdatedTime types.Time `json:"updatedTime"`
	List        []ADLAlert `json:"list"`
}

// ADLAlert holds a contract's auto-deleveraging (ADL) thresholds and the state of its insurance pool
type ADLAlert struct {
	Coin   currency.Code `json:"coin"`
	Symbol string        `json:"symbol"`
	// Balance is the insurance pool balance; a shared pool's balance refreshes daily at 00:00 UTC
	Balance types.Number `json:"balance"`
	// MaxBalance is deprecated and always empty
	MaxBalance types.Number `json:"maxBalance"`
	// InsurancePNLRatio is the 8 hour PnL drawdown ratio below which ADL triggers
	InsurancePNLRatio types.Number `json:"insurancePnlRatio"`
	// PNLRatio is the symbol's PnL drawdown ratio over the last 8 hours
	PNLRatio types.Number `json:"pnlRatio"`
	// ADLTriggerThreshold is the insurance pool balance above which PnL drawdown ADL can trigger
	ADLTriggerThreshold types.Number `json:"adlTriggerThreshold"`
	// ADLStopRatio is the PnL drawdown ratio above which ADL stops
	ADLStopRatio types.Number `json:"adlStopRatio"`
}

// FeeGroupStructureResponse holds contract fee groups
type FeeGroupStructureResponse struct {
	List []FeeGroup `json:"list"`
}

// FeeGroup holds a contract fee group's symbols and the fee rates of each Pro and market maker level
type FeeGroup struct {
	GroupName       string        `json:"groupName"`
	WeightingFactor uint64        `json:"weightingFactor"`
	SymbolsNumbers  uint64        `json:"symbolsNumbers"`
	Symbols         []string      `json:"symbols"`
	FeeRates        FeeGroupRates `json:"feeRates"`
	UpdateTime      types.Time    `json:"updateTime"`
}

// FeeGroupRates holds a fee group's rates by client level
type FeeGroupRates struct {
	Pro         []ProFeeRate         `json:"pro"`
	MarketMaker []MarketMakerFeeRate `json:"marketMaker"`
}

// ProFeeRate holds a Pro level's fee rates
type ProFeeRate struct {
	Level        string       `json:"level"`
	TakerFeeRate types.Number `json:"takerFeeRate"`
	MakerFeeRate types.Number `json:"makerFeeRate"`
}

// MarketMakerFeeRate holds a market maker level's rebate rates
type MarketMakerFeeRate struct {
	Level          string       `json:"level"`
	MakerRebate    types.Number `json:"makerRebate"`
	RPIMakerRebate types.Number `json:"rpiMakerRebate"`
}

// OptionBaseCoinsResponse holds option base coins
type OptionBaseCoinsResponse struct {
	List []OptionBaseCoin `json:"list"`
}

// OptionBaseCoin holds an option base coin
type OptionBaseCoin struct {
	BaseCoin         currency.Code `json:"baseCoin"`
	QuoteCoin        currency.Code `json:"quoteCoin"`
	SettleCoin       currency.Code `json:"settleCoin"`
	OptionShowName   string        `json:"optionShowName"`
	OptionOnlineTime types.Time    `json:"optionOnlineTime"`
	// HasSymbol is false until the base coin has tradable option symbols
	HasSymbol types.Boolean `json:"hasSymbol"`
	// UnderlyingType is 0 for crypto, 1 commodity, 2 stock, 3 forex and 4 oil
	UnderlyingType uint64 `json:"underlyingType"`
}

// TickerCommon holds the fields of a Bybit ticker, of which tickerVolumes reads the two volume figures
type TickerCommon struct {
	Symbol                     string       `json:"symbol"`
	TickDirection              string       `json:"tickDirection"`
	LastPrice                  types.Number `json:"lastPrice"`
	IndexPrice                 types.Number `json:"indexPrice"`
	MarkPrice                  types.Number `json:"markPrice"`
	PreviousPrice24Hour        types.Number `json:"prevPrice24h"`
	Price24HourPercent         types.Number `json:"price24hPcnt"`
	HighPrice24Hour            types.Number `json:"highPrice24h"`
	LowPrice24Hour             types.Number `json:"lowPrice24h"`
	PreviousPrice1Hour         types.Number `json:"prevPrice1h"`
	OpenInterest               types.Number `json:"openInterest"`
	OpenInterestValue          types.Number `json:"openInterestValue"`
	Turnover24Hour             types.Number `json:"turnover24h"`
	Volume24Hour               types.Number `json:"volume24h"`
	FundingRate                types.Number `json:"fundingRate"`
	NextFundingTime            types.Time   `json:"nextFundingTime"`
	PredictedDeliveryPrice     types.Number `json:"predictedDeliveryPrice"`
	BasisRate                  types.Number `json:"basisRate"`
	DeliveryFeeRate            types.Number `json:"deliveryFeeRate"`
	DeliveryTime               types.Time   `json:"deliveryTime"`
	Ask1Size                   types.Number `json:"ask1Size"`
	Bid1Price                  types.Number `json:"bid1Price"`
	Ask1Price                  types.Number `json:"ask1Price"`
	Bid1Size                   types.Number `json:"bid1Size"`
	Basis                      types.Number `json:"basis"`
	Bid1ImpliedVolatility      types.Number `json:"bid1Iv"`
	Ask1ImpliedVolatility      types.Number `json:"ask1Iv"`
	MarkImpliedVolatility      types.Number `json:"markIv"`
	MarkPriceImpliedVolatility types.Number `json:"markPriceIv"`
	UnderlyingPrice            types.Number `json:"underlyingPrice"`
	TotalVolume                types.Number `json:"totalVolume"`
	TotalTurnover              types.Number `json:"totalTurnover"`
	Delta                      types.Number `json:"delta"`
	Gamma                      types.Number `json:"gamma"`
	Vega                       types.Number `json:"vega"`
	Theta                      types.Number `json:"theta"`
	Change24Hour               types.Number `json:"change24h"`
	USDIndexPrice              types.Number `json:"usdIndexPrice"`
	BidPrice                   types.Number `json:"bidPrice"`
	BidSize                    types.Number `json:"bidSize"`
	BidImpliedVolatility       types.Number `json:"bidIv"`
	AskPrice                   types.Number `json:"askPrice"`
	AskSize                    types.Number `json:"askSize"`
	AskImpliedVolatility       types.Number `json:"askIv"`
}
