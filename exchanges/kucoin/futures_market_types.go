package kucoin

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Contract holds a futures contract's specification and market state (Get Symbol, Get All Symbols)
type Contract struct {
	Symbol        string        `json:"symbol"`
	DisplaySymbol string        `json:"displaySymbol"`
	RootSymbol    currency.Code `json:"rootSymbol"`
	// Type is FFWCSX for perpetual contracts and FFICSX for dated futures
	Type          string        `json:"type"`
	FirstOpenDate types.Time    `json:"firstOpenDate"`
	ExpireDate    types.Time    `json:"expireDate"`
	SettleDate    types.Time    `json:"settleDate"`
	BaseCurrency  currency.Code `json:"baseCurrency"`
	// DisplayBaseCurrency is the name the web and app show, which can differ from BaseCurrency, for example in CJK
	DisplayBaseCurrency    string        `json:"displayBaseCurrency"`
	QuoteCurrency          currency.Code `json:"quoteCurrency"`
	SettleCurrency         currency.Code `json:"settleCurrency"`
	MaxOrderQuantity       uint64        `json:"maxOrderQty"`
	MarketMaxOrderQuantity uint64        `json:"marketMaxOrderQty"`
	MaxPrice               float64       `json:"maxPrice"`
	LotSize                uint64        `json:"lotSize"`
	TickSize               float64       `json:"tickSize"`
	IndexPriceTickSize     float64       `json:"indexPriceTickSize"`
	// Multiplier is the base currency amount of one lot; coin-margined contracts, whose lot is worth 1 USD, report -1
	Multiplier     float64 `json:"multiplier"`
	InitialMargin  float64 `json:"initialMargin"`
	MaintainMargin float64 `json:"maintainMargin"`
	MaxRiskLimit   uint64  `json:"maxRiskLimit"`
	MinRiskLimit   uint64  `json:"minRiskLimit"`
	RiskStep       uint64  `json:"riskStep"`
	MakerFeeRate   float64 `json:"makerFeeRate"`
	TakerFeeRate   float64 `json:"takerFeeRate"`
	// TakerFixFee and MakerFixFee are deprecated
	TakerFixFee             float64 `json:"takerFixFee"`
	MakerFixFee             float64 `json:"makerFixFee"`
	SettlementFee           float64 `json:"settlementFee"`
	IsDeleverage            bool    `json:"isDeleverage"`
	IsQuanto                bool    `json:"isQuanto"` // deprecated
	IsInverse               bool    `json:"isInverse"`
	MarkMethod              string  `json:"markMethod"`
	FairMethod              string  `json:"fairMethod"`
	FundingBaseSymbol       string  `json:"fundingBaseSymbol"`
	FundingQuoteSymbol      string  `json:"fundingQuoteSymbol"`
	FundingRateSymbol       string  `json:"fundingRateSymbol"`
	IndexSymbol             string  `json:"indexSymbol"`
	SettlementSymbol        string  `json:"settlementSymbol"`
	Status                  string  `json:"status"`
	FundingFeeRate          float64 `json:"fundingFeeRate"`
	PredictedFundingFeeRate float64 `json:"predictedFundingFeeRate"`
	// DailyInterestRate is sent but not documented
	DailyInterestRate float64 `json:"dailyInterestRate"`
	// FundingRateGranularity is the configured funding interval in milliseconds; a change applies from the next cycle,
	// so it can differ from CurrentFundingRateGranularity until then
	FundingRateGranularity uint64  `json:"fundingRateGranularity"`
	FundingRateCap         float64 `json:"fundingRateCap"`
	FundingRateFloor       float64 `json:"fundingRateFloor"`
	// Period is the deprecated funding fee collection mode: 0 deferred, 1 current period
	Period                             uint64     `json:"period"`
	EffectiveFundingRateCycleStartTime types.Time `json:"effectiveFundingRateCycleStartTime"`
	// CurrentFundingRateGranularity is the funding interval in effect, in milliseconds
	CurrentFundingRateGranularity uint64       `json:"currentFundingRateGranularity"`
	OpenInterest                  types.Number `json:"openInterest"` // lots
	TurnoverOf24Hour              float64      `json:"turnoverOf24h"`
	VolumeOf24Hour                float64      `json:"volumeOf24h"`
	MarkPrice                     float64      `json:"markPrice"`
	IndexPrice                    float64      `json:"indexPrice"`
	LastTradePrice                float64      `json:"lastTradePrice"`
	// NextFundingRateTime is the time left until the next funding settlement in milliseconds, not a timestamp; as the
	// difference between two times it stays signed
	NextFundingRateTime       int64      `json:"nextFundingRateTime"`
	NextFundingRateDateTime   types.Time `json:"nextFundingRateDateTime"`
	MaxLeverage               uint64     `json:"maxLeverage"`
	SourceExchanges           []string   `json:"sourceExchanges"`
	PremiumsSymbol1Minute     string     `json:"premiumsSymbol1M"`
	PremiumsSymbol8Hour       string     `json:"premiumsSymbol8H"`
	FundingBaseSymbol1Minute  string     `json:"fundingBaseSymbol1M"`
	FundingQuoteSymbol1Minute string     `json:"fundingQuoteSymbol1M"`
	LowPrice                  float64    `json:"lowPrice"`
	HighPrice                 float64    `json:"highPrice"`
	PriceChangePercentage     float64    `json:"priceChgPct"`
	PriceChange               float64    `json:"priceChg"`
	// K, M and F shape the risk limit curve: K amplifies the maximum open size, M is the margin curve slope and F
	// multiplies the maintenance margin rate into the initial margin rate
	K                   float64 `json:"k"`
	M                   float64 `json:"m"`
	F                   float64 `json:"f"`
	MMRLimit            float64 `json:"mmrLimit"`
	MMRLeverageConstant float64 `json:"mmrLevConstant"`
	SupportCross        bool    `json:"supportCross"`
	BuyLimit            float64 `json:"buyLimit"`
	SellLimit           float64 `json:"sellLimit"`
	// AdjustK, AdjustM, AdjustMMRLeverageConstant and AdjustActiveTime describe risk limit parameters waiting to take
	// effect and are null while none are pending
	AdjustK                   float64    `json:"adjustK"`
	AdjustM                   float64    `json:"adjustM"`
	AdjustMMRLeverageConstant float64    `json:"adjustMmrLevConstant"`
	AdjustActiveTime          types.Time `json:"adjustActiveTime"`
	CrossRiskLimit            float64    `json:"crossRiskLimit"`
	MarketStage               string     `json:"marketStage"`
	PreMarketToPerpDate       types.Time `json:"preMarketToPerpDate"`
	OrderPriceRange           float64    `json:"orderPriceRange"`
	MarketType                string     `json:"marketType"` // deprecated in favour of AssetClass and SubMarketType
	AssetClass                string     `json:"assetClass"`
	SubMarketType             string     `json:"subMarketType"`
	// LastTimeFundingRate is sent but not documented
	LastTimeFundingRate float64 `json:"lastTimeFundingRate"`
}

// FuturesContractResponse holds a futures contract's specification and market state (Get Symbol)
type FuturesContractResponse Contract

// FuturesTicker holds a contract's last fill and best bid and ask (Get Ticker, Get All Tickers)
type FuturesTicker struct {
	Sequence     uint64       `json:"sequence"`
	Symbol       string       `json:"symbol"`
	Side         string       `json:"side"` // taker side of the last fill
	Size         uint64       `json:"size"` // lots in the last fill
	TradeID      string       `json:"tradeId"`
	Price        types.Number `json:"price"`
	BestBidPrice types.Number `json:"bestBidPrice"`
	BestBidSize  uint64       `json:"bestBidSize"`
	BestAskPrice types.Number `json:"bestAskPrice"`
	BestAskSize  uint64       `json:"bestAskSize"`
	Timestamp    types.Time   `json:"ts"` // time of the last fill
}

// FuturesTickerResponse holds a contract's last fill and best bid and ask (Get Ticker)
type FuturesTickerResponse FuturesTicker

// FuturesOrderbookResponse holds a contract's order book aggregated by price, bids from high to low and asks from low
// to high (Get Full OrderBook, Get Part OrderBook)
type FuturesOrderbookResponse struct {
	Sequence  uint64                           `json:"sequence"`
	Symbol    string                           `json:"symbol"`
	Bids      orderbook.LevelsArrayPriceAmount `json:"bids"`
	Asks      orderbook.LevelsArrayPriceAmount `json:"asks"`
	Timestamp types.Time                       `json:"ts"`
}

// FuturesTrade holds one of a contract's fills (Get Trade History)
type FuturesTrade struct {
	Sequence     uint64       `json:"sequence"`
	ContractID   uint64       `json:"contractId"` // deprecated
	TradeID      string       `json:"tradeId"`
	MakerOrderID string       `json:"makerOrderId"`
	TakerOrderID string       `json:"takerOrderId"`
	Timestamp    types.Time   `json:"ts"`
	Size         uint64       `json:"size"` // lots
	Price        types.Number `json:"price"`
	Side         string       `json:"side"` // taker side
}

// FuturesKlineRequest holds the parameters of Get Klines
type FuturesKlineRequest struct {
	// Symbol is a contract, index or premium index symbol
	Symbol string
	// Granularity is sent in whole minutes
	Granularity kline.Interval
	From        time.Time
	To          time.Time
}

// FuturesKline holds one candle (Get Klines)
type FuturesKline struct {
	StartTime types.Time
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64 // lots, counting one side of each trade
	Turnover  float64 // quote currency, counting one side of each trade
}

// UnmarshalJSON decodes a candle from KuCoin's array of start time, open, high, low, close, volume and turnover
func (f *FuturesKline) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[7]any{&f.StartTime, &f.Open, &f.High, &f.Low, &f.Close, &f.Volume, &f.Turnover})
}

// FuturesMarkPriceResponse holds a contract's current mark price (Get Mark Price)
type FuturesMarkPriceResponse struct {
	Symbol      string     `json:"symbol"`
	Granularity uint64     `json:"granularity"` // milliseconds
	TimePoint   types.Time `json:"timePoint"`
	Value       float64    `json:"value"` // mark price
	IndexPrice  float64    `json:"indexPrice"`
}

// FuturesHistoryRequest holds the parameters of Get Spot Index Price, Get Interest Rate Index, Get Premium Index and
// Get Private Funding History
type FuturesHistoryRequest struct {
	Symbol  string
	StartAt time.Time
	EndAt   time.Time
	// Reverse and Forward default to true on the server, so nil leaves them unsent and false must be explicit
	Reverse *bool
	// Offset is the unique attribute of the last record the previous request returned
	Offset   uint64
	Forward  *bool
	MaxCount uint64
}

// FuturesSpotIndexPriceResponse holds a page of a spot index's prices (Get Spot Index Price)
type FuturesSpotIndexPriceResponse struct {
	DataList []FuturesSpotIndexPrice `json:"dataList"`
	HasMore  bool                    `json:"hasMore"`
}

// FuturesSpotIndexPrice holds a spot index's price at one time point and each exchange's contribution to it
type FuturesSpotIndexPrice struct {
	Symbol          string                      `json:"symbol"`
	Granularity     uint64                      `json:"granularity"` // milliseconds
	TimePoint       types.Time                  `json:"timePoint"`
	Value           float64                     `json:"value"`
	DecomposionList []FuturesSpotIndexComponent `json:"decomposionList"`
}

// FuturesSpotIndexComponent holds one exchange's price and weight in a spot index
type FuturesSpotIndexComponent struct {
	Exchange     string  `json:"exchange"`
	ExchangeName string  `json:"exchangeName"` // sent but not documented
	Price        float64 `json:"price"`
	Weight       float64 `json:"weight"`
}

// FuturesIndexHistoryResponse holds a page of an index's values (Get Interest Rate Index, Get Premium Index)
type FuturesIndexHistoryResponse struct {
	DataList []FuturesIndexValue `json:"dataList"`
	HasMore  bool                `json:"hasMore"`
}

// FuturesIndexValue holds an interest rate or premium index value at one time point
type FuturesIndexValue struct {
	Symbol      string     `json:"symbol"`
	Granularity uint64     `json:"granularity"` // milliseconds
	TimePoint   types.Time `json:"timePoint"`
	Value       float64    `json:"value"`
}

// Futures24HourStatsResponse holds the platform's futures turnover over the last 24 hours (Get 24hr stats)
type Futures24HourStatsResponse struct {
	TurnoverOf24Hour float64 `json:"turnoverOf24h"` // USD, counting one side of each trade
}

// FuturesServiceStatusResponse holds the futures service status (Get Service Status)
type FuturesServiceStatusResponse struct {
	Message string `json:"msg"`
	// Status is open for normal trading, close for maintenance and cancelonly while orders can only be cancelled
	Status string `json:"status"`
}
