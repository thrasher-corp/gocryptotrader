package binanceus

import (
	"bytes"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// objectOrArray decodes a response that is a single object when one symbol is requested and an array otherwise
type objectOrArray[T any] []T

// UnmarshalJSON decodes a JSON object into a one-element slice and a JSON array into a slice
func (o *objectOrArray[T]) UnmarshalJSON(data []byte) error {
	if trimmed := bytes.TrimLeft(data, " \t\r\n"); len(trimmed) > 0 && trimmed[0] == '{' {
		var single T
		if err := json.Unmarshal(trimmed, &single); err != nil {
			return err
		}
		*o = objectOrArray[T]{single}
		return nil
	}
	return json.Unmarshal(data, (*[]T)(o))
}

// RecentTradeRequest holds the parameters of a recent trades request
type RecentTradeRequest struct {
	Symbol currency.Pair
	Limit  uint64 // 500 when zero, at most 1000
}

// RecentTradeResponse holds a trade of the recent or historical trades lists
type RecentTradeResponse struct {
	ID            uint64       `json:"id"`
	Price         types.Number `json:"price"`
	Quantity      types.Number `json:"qty"`
	QuoteQuantity types.Number `json:"quoteQty"`
	Time          types.Time   `json:"time"`
	IsBuyerMaker  bool         `json:"isBuyerMaker"`
	IsBestMatch   bool         `json:"isBestMatch"`
}

// AggregatedTradeRequest holds the parameters of an aggregate trades request. FromID cannot be combined with StartTime
// or EndTime, and a Limit above 1000 is collected over several requests, which needs FromID or StartTime
type AggregatedTradeRequest struct {
	Symbol    currency.Pair
	FromID    uint64
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
}

// AggregatedTradeResponse holds an aggregate trade: the fills of one taker order at one price
type AggregatedTradeResponse struct {
	AggregateTradeID uint64       `json:"a"`
	Price            types.Number `json:"p"`
	Quantity         types.Number `json:"q"`
	FirstTradeID     uint64       `json:"f"`
	LastTradeID      uint64       `json:"l"`
	Timestamp        types.Time   `json:"T"`
	IsBuyerMaker     bool         `json:"m"`
	IsBestMatch      bool         `json:"M"`
}

// OrderBookRequest holds the parameters of an order book request
type OrderBookRequest struct {
	Symbol currency.Pair
	// Limit is the number of levels on each side, 100 when zero and at most 5000
	Limit uint64
	// SymbolStatus rejects the request when the symbol's trading status differs
	SymbolStatus SymbolStatus
}

// OrderBookResponse holds an order book
type OrderBookResponse struct {
	LastUpdateID int64                            `json:"lastUpdateId"` // int64 to match the order book API it feeds
	Bids         orderbook.LevelsArrayPriceAmount `json:"bids"`
	Asks         orderbook.LevelsArrayPriceAmount `json:"asks"`
}

// KlinesRequest holds the parameters of a klines request
type KlinesRequest struct {
	Symbol   currency.Pair
	Interval kline.Interval
	Limit    uint64 // 500 when zero, at most 1000
	// StartTime and EndTime are always in UTC
	StartTime time.Time
	EndTime   time.Time
	// TimeZone sets the time zone the intervals are interpreted in, as hours and minutes or hours alone from -12:00 to
	// +14:00; UTC when empty
	TimeZone string
}

// CandleStickResponse holds a kline, a candlestick bar
type CandleStickResponse struct {
	OpenTime                 types.Time
	OpenPrice                types.Number
	HighPrice                types.Number
	LowPrice                 types.Number
	ClosePrice               types.Number
	Volume                   types.Number
	CloseTime                types.Time
	QuoteAssetVolume         types.Number
	NumberOfTrades           uint64
	TakerBuyBaseAssetVolume  types.Number
	TakerBuyQuoteAssetVolume types.Number
}

// UnmarshalJSON decodes a kline from its array form, whose twelfth element is unused
func (c *CandleStickResponse) UnmarshalJSON(data []byte) error {
	target := [12]any{&c.OpenTime, &c.OpenPrice, &c.HighPrice, &c.LowPrice, &c.ClosePrice, &c.Volume, &c.CloseTime, &c.QuoteAssetVolume, &c.NumberOfTrades, &c.TakerBuyBaseAssetVolume, &c.TakerBuyQuoteAssetVolume, nil}
	return json.Unmarshal(data, &target)
}

// SymbolPriceResponse holds a symbol's latest price
type SymbolPriceResponse struct {
	Symbol string       `json:"symbol"`
	Price  types.Number `json:"price"`
}

// AveragePriceResponse holds a symbol's current average price
type AveragePriceResponse struct {
	IntervalInMinutes uint64       `json:"mins"`
	Price             types.Number `json:"price"`
	CloseTime         types.Time   `json:"closeTime"` // Time of the last trade
}

// BookTickerResponse holds a symbol's best bid and ask price and quantity
type BookTickerResponse struct {
	Symbol      string       `json:"symbol"`
	BidPrice    types.Number `json:"bidPrice"`
	BidQuantity types.Number `json:"bidQty"`
	AskPrice    types.Number `json:"askPrice"`
	AskQuantity types.Number `json:"askQty"`
}

// PriceChangeStatsRequest holds the parameters of a 24hr ticker price change statistics request
type PriceChangeStatsRequest struct {
	Symbols      currency.Pairs // Every symbol when empty
	Type         TickerType     // FULL when empty, or MINI
	SymbolStatus SymbolStatus
}

// PriceChangeStatsResponse holds 24hr ticker price change statistics; the MINI type leaves out the price change,
// previous close, last quantity, bid and ask fields
type PriceChangeStatsResponse struct {
	Symbol               string       `json:"symbol"`
	PriceChange          types.Number `json:"priceChange"`
	PriceChangePercent   types.Number `json:"priceChangePercent"`
	WeightedAveragePrice types.Number `json:"weightedAvgPrice"`
	PreviousClosePrice   types.Number `json:"prevClosePrice"`
	LastPrice            types.Number `json:"lastPrice"`
	LastQuantity         types.Number `json:"lastQty"`
	BidPrice             types.Number `json:"bidPrice"`
	BidQuantity          types.Number `json:"bidQty"`
	AskPrice             types.Number `json:"askPrice"`
	AskQuantity          types.Number `json:"askQty"`
	OpenPrice            types.Number `json:"openPrice"`
	HighPrice            types.Number `json:"highPrice"`
	LowPrice             types.Number `json:"lowPrice"`
	Volume               types.Number `json:"volume"`
	QuoteVolume          types.Number `json:"quoteVolume"`
	OpenTime             types.Time   `json:"openTime"`
	CloseTime            types.Time   `json:"closeTime"`
	FirstID              int64        `json:"firstId"` // -1 when there were no trades
	LastID               int64        `json:"lastId"`  // -1 when there were no trades
	Count                uint64       `json:"count"`
}

// RollingWindowTickerRequest holds the parameters of a rolling window price change statistics request
type RollingWindowTickerRequest struct {
	Symbols currency.Pairs // At least one and at most 100
	// WindowSize is sent as whole minutes from 1 to 59, hours from 1 to 23 or days from 1 to 7; one day when zero
	WindowSize time.Duration
	Type       TickerType // FULL when empty, or MINI
	// SymbolStatus filters the symbols by trading status; the REST endpoint takes it though only its WebSocket API form
	// documents it
	SymbolStatus SymbolStatus
}

// TickerStatisticsResponse holds a symbol's price change statistics over a trading day or a rolling window. The MINI
// type leaves out the price change fields
type TickerStatisticsResponse struct {
	Symbol               string       `json:"symbol"`
	PriceChange          types.Number `json:"priceChange"`
	PriceChangePercent   types.Number `json:"priceChangePercent"`
	WeightedAveragePrice types.Number `json:"weightedAvgPrice"`
	OpenPrice            types.Number `json:"openPrice"`
	HighPrice            types.Number `json:"highPrice"`
	LowPrice             types.Number `json:"lowPrice"`
	LastPrice            types.Number `json:"lastPrice"`
	Volume               types.Number `json:"volume"`
	QuoteVolume          types.Number `json:"quoteVolume"`
	OpenTime             types.Time   `json:"openTime"`
	CloseTime            types.Time   `json:"closeTime"`
	FirstID              int64        `json:"firstId"` // -1 when there were no trades
	LastID               int64        `json:"lastId"`  // -1 when there were no trades
	Count                uint64       `json:"count"`
}

// TradingDayTickerRequest holds the parameters of a trading day ticker request
type TradingDayTickerRequest struct {
	Symbols currency.Pairs // At least one and at most 100
	// TimeZone sets the time zone the trading day starts in, as hours and minutes or hours alone; UTC when empty
	TimeZone     string
	Type         TickerType // FULL when empty, or MINI
	SymbolStatus SymbolStatus
}
