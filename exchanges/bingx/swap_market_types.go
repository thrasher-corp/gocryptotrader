package bingx

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// SwapContractResponse holds a perpetual contract from USDT-M Perp Futures symbols
type SwapContractResponse struct {
	ContractID        string             `json:"contractId"`
	Symbol            string             `json:"symbol"`
	Size              types.Number       `json:"size"`
	QuantityPrecision uint64             `json:"quantityPrecision"`
	PricePrecision    uint64             `json:"pricePrecision"`
	FeeRate           float64            `json:"feeRate"`
	MakerFeeRate      float64            `json:"makerFeeRate"`
	TakerFeeRate      float64            `json:"takerFeeRate"`
	TradeMinLimit     float64            `json:"tradeMinLimit"`
	TradeMinQuantity  float64            `json:"tradeMinQuantity"` // in the traded asset
	TradeMinUSDT      float64            `json:"tradeMinUSDT"`
	Currency          currency.Code      `json:"currency"` // settlement and margin currency, USDT or USDC
	Asset             currency.Code      `json:"asset"`
	Status            SwapContractStatus `json:"status"`
	APIStateOpen      types.Boolean      `json:"apiStateOpen"`  // whether positions can be opened through the API
	APIStateClose     types.Boolean      `json:"apiStateClose"` // whether positions can be closed through the API
	EnsureTrigger     bool               `json:"ensureTrigger"` // whether guaranteed stop losses are supported
	TriggerFeeRate    types.Number       `json:"triggerFeeRate"`
	BrokerState       bool               `json:"brokerState"`  // true when broker users may not trade the contract
	LaunchTime        types.Time         `json:"launchTime"`   // when the contract goes online
	MaintainTime      types.Time         `json:"maintainTime"` // when opening positions stops; zero when not set
	OffTime           types.Time         `json:"offTime"`      // delisting time; zero when not set
	DisplayName       string             `json:"displayName"`  // the name shown on the platform, which TradFi contracts differ in
}

// SwapOrderBookResponse holds a perpetual contract's order book from Order Book
type SwapOrderBookResponse struct {
	Time types.Time                       `json:"T"`
	Bids orderbook.LevelsArrayPriceAmount `json:"bids"`
	Asks orderbook.LevelsArrayPriceAmount `json:"asks"`
	// BidsCoin and AsksCoin hold the same levels with their quantities in the traded asset
	BidsCoin orderbook.LevelsArrayPriceAmount `json:"bidsCoin"`
	AsksCoin orderbook.LevelsArrayPriceAmount `json:"asksCoin"`
}

// SwapRecentTradeResponse holds a market trade from Recent Trades List
type SwapRecentTradeResponse struct {
	Time          types.Time   `json:"time"`
	IsBuyerMaker  bool         `json:"isBuyerMaker"`
	Price         types.Number `json:"price"`
	Quantity      types.Number `json:"qty"`
	QuoteQuantity types.Number `json:"quoteQty"`
	FillID        string       `json:"fillId"` // not documented, but sent live
	Timestamp     types.Time   `json:"ts"`     // not documented, but sent live
}

// SwapMarkPriceAndFundingRateResponse holds a perpetual contract's mark price and funding rate from Mark Price and
// Funding Rate
type SwapMarkPriceAndFundingRateResponse struct {
	Symbol               string       `json:"symbol"`
	MarkPrice            types.Number `json:"markPrice"`
	IndexPrice           types.Number `json:"indexPrice"`
	LastFundingRate      types.Number `json:"lastFundingRate"`
	NextFundingTime      types.Time   `json:"nextFundingTime"`
	FundingIntervalHours uint64       `json:"fundingIntervalHours"`
	MinFundingRate       types.Number `json:"minFundingRate"`
	MaxFundingRate       types.Number `json:"maxFundingRate"`
	UpdateTime           types.Time   `json:"updateTime"`
}

// SwapFundingRateHistoryRequest holds the parameters of Get Funding Rate History
type SwapFundingRateHistoryRequest struct {
	Symbol    currency.Pair
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64 // 100 when zero, at most 1000
}

// SwapFundingRateHistoryResponse holds a completed funding settlement from Get Funding Rate History
type SwapFundingRateHistoryResponse struct {
	Symbol      string       `json:"symbol"`
	FundingRate types.Number `json:"fundingRate"`
	FundingTime types.Time   `json:"fundingTime"`
	MarkPrice   types.Number `json:"markPrice"` // not documented, but sent live
}

// SwapKlinesRequest holds the parameters of Kline/Candlestick Data
type SwapKlinesRequest struct {
	Symbol    currency.Pair
	Interval  kline.Interval
	StartTime time.Time
	EndTime   time.Time
	// TimeZone is the UTC offset in hours that candles of a day or longer open at: 0, the default, or 8
	TimeZone uint64
	Limit    uint64 // 500 when zero, at most 1440
}

// SwapKlineResponse holds a candle from Kline/Candlestick Data
type SwapKlineResponse struct {
	Open   types.Number `json:"open"`
	Close  types.Number `json:"close"`
	High   types.Number `json:"high"`
	Low    types.Number `json:"low"`
	Volume types.Number `json:"volume"`
	Time   types.Time   `json:"time"` // open time
}

// SwapOpenInterestResponse holds a perpetual contract's open interest from Open Interest Statistics
type SwapOpenInterestResponse struct {
	OpenInterest types.Number `json:"openInterest"`
	Symbol       string       `json:"symbol"`
	Time         types.Time   `json:"time"`
}

// Swap24HourTickerResponse holds a perpetual contract's price change statistics over 24 hours from 24hr Ticker Price
// Change Statistics
type Swap24HourTickerResponse struct {
	Symbol             string       `json:"symbol"`
	PriceChange        types.Number `json:"priceChange"`
	PriceChangePercent types.Number `json:"priceChangePercent"`
	LastPrice          types.Number `json:"lastPrice"`
	LastQuantity       types.Number `json:"lastQty"`
	HighPrice          types.Number `json:"highPrice"`
	LowPrice           types.Number `json:"lowPrice"`
	Volume             types.Number `json:"volume"`
	QuoteVolume        types.Number `json:"quoteVolume"`
	OpenPrice          types.Number `json:"openPrice"`
	OpenTime           types.Time   `json:"openTime"`
	CloseTime          types.Time   `json:"closeTime"`
	BidPrice           types.Number `json:"bidPrice"`
	BidQuantity        types.Number `json:"bidQty"`
	AskPrice           types.Number `json:"askPrice"`
	AskQuantity        types.Number `json:"askQty"`
}

// SwapHistoricalTradesRequest holds the parameters of Query historical transaction orders
type SwapHistoricalTradesRequest struct {
	Symbol currency.Pair
	FromID uint64 // the trade ID to start from; the most recent trades when zero
	Limit  uint64 // 50 when zero, at most 100
}

// SwapHistoricalTradeResponse holds a market trade from Query historical transaction orders
type SwapHistoricalTradeResponse struct {
	Time          types.Time   `json:"time"`
	IsBuyerMaker  bool         `json:"isBuyerMaker"`
	Price         types.Number `json:"price"`
	Quantity      types.Number `json:"qty"`
	QuoteQuantity types.Number `json:"quoteQty"`
	ID            string       `json:"id"`
}

// SwapOrderBookTickerResponse holds the reply of Symbol Order Book Ticker
type SwapOrderBookTickerResponse struct {
	BookTicker *SwapOrderBookTicker `json:"book_ticker"`
}

// SwapOrderBookTicker holds a perpetual contract's best bid and ask
type SwapOrderBookTicker struct {
	Symbol       string     `json:"symbol"`
	BidPrice     float64    `json:"bid_price"`
	BidQuantity  float64    `json:"bid_qty"`
	AskPrice     float64    `json:"ask_price"`
	AskQuantity  float64    `json:"ask_qty"`
	LastUpdateID uint64     `json:"lastUpdateId"`
	Time         types.Time `json:"time"`
}

// SwapMarkPriceKlinesRequest holds the parameters of Mark Price Kline/Candlestick Data
type SwapMarkPriceKlinesRequest struct {
	Symbol    currency.Pair
	Interval  kline.Interval
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64 // 500 when zero, at most 1440
}

// SwapMarkPriceKlineResponse holds a mark price candle from Mark Price Kline/Candlestick Data
type SwapMarkPriceKlineResponse struct {
	Open      types.Number `json:"open"`
	Close     types.Number `json:"close"`
	High      types.Number `json:"high"`
	Low       types.Number `json:"low"`
	Volume    types.Number `json:"volume"`
	OpenTime  types.Time   `json:"openTime"`
	CloseTime types.Time   `json:"closeTime"` // the candle's end, which does not mark it final
}

// SwapPriceTickerResponse holds a perpetual contract's latest price from Symbol Price Ticker
type SwapPriceTickerResponse struct {
	Symbol string       `json:"symbol"`
	Price  types.Number `json:"price"`
	Time   types.Time   `json:"time"`
}

// SwapTradingRulesResponse holds a perpetual contract's trading rules from Trading Rules. The buy and sell price
// limits are ratios to the current price that a limit order's price must lie between
type SwapTradingRulesResponse struct {
	MinSizeCoin    types.Number `json:"minSizeCoin"`
	MinSizeUSD     types.Number `json:"minSizeUsd"`
	MaxNumberOrder types.Number `json:"maxNumOrder"` // open orders allowed on the contract, conditional orders included
	// ProtectionThreshold is the largest gap between the latest and mark prices at which a triggered order is placed
	// while spread protection is on
	ProtectionThreshold types.Number `json:"protectionThreshold"`
	BuyMaxPrice         types.Number `json:"buyMaxPrice"`
	BuyMinPrice         types.Number `json:"buyMinPrice"`
	SellMaxPrice        types.Number `json:"sellMaxPrice"`
	SellMinPrice        types.Number `json:"sellMinPrice"`
	// MarketRatio is the largest gap between a market order's execution price and the mark price, as a ratio
	MarketRatio types.Number `json:"marketRatio"`
}

// SwapTradingScheduleResponse holds a non-crypto perpetual contract's trading schedule from Trading Schedule
type SwapTradingScheduleResponse struct {
	Symbol       string                `json:"symbol"`
	ContractType NonCryptoContractType `json:"contractType"`
	MarketStatus MarketStatus          `json:"marketStatus"`
	TimeZone     string                `json:"timeZone"`   // the trading calendar's time zone, UTC+08:00
	UpdateTime   types.Time            `json:"updateTime"` // when the contract's configuration changed; zero when unknown
	Sessions     []SwapTradingSession  `json:"sessions"`
}

// SwapTradingSession holds a scheduled trading or non-trading interval, which includes its start and excludes its end
type SwapTradingSession struct {
	StartTime types.Time         `json:"startTime"`
	EndTime   types.Time         `json:"endTime"`
	Type      TradingSessionType `json:"type"`
}
