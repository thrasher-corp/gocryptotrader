package kraken

import (
	"errors"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/types"
)

var (
	errTickTypeEmpty      = errors.New("tick type is empty")
	errAnalyticsTypeEmpty = errors.New("analytics type is empty")
)

// FuturesCandlesRequest holds the parameters of Market Candles
type FuturesCandlesRequest struct {
	// TickType is spot, mark or trade, as GetFuturesTickTypes lists
	TickType string
	Pair     currency.Pair
	// Interval is 1, 5, 15 or 30 minutes, 1, 4 or 12 hours, 1 day or 1 week
	Interval kline.Interval
	// From returns the candles from it rather than the most recent ones
	From time.Time
	// To returns the candles up to it rather than up to now
	To time.Time
	// Count is the number of candles to return, 2000 when it is 0
	Count uint64
}

// FuturesCandlesResponse holds a market's candles in ascending time order
type FuturesCandlesResponse struct {
	Candles []FuturesCandle `json:"candles"`
	// MoreCandles is set when the range holds more candles after the last one returned
	MoreCandles bool `json:"more_candles"`
}

// FuturesCandle is an OHLC candle
type FuturesCandle struct {
	Time   types.Time   `json:"time"`
	Open   types.Number `json:"open"`
	High   types.Number `json:"high"`
	Low    types.Number `json:"low"`
	Close  types.Number `json:"close"`
	Volume types.Number `json:"volume"`
}

// FuturesLiquidityPoolStatisticRequest holds the parameters of Get liquidity pool statistic
type FuturesLiquidityPoolStatisticRequest struct {
	Since time.Time
	// To returns the buckets up to it rather than up to now
	To time.Time
	// Interval is the bucket size: 1, 5, 15 or 30 minutes, 1, 4 or 12 hours, 1 day or 1 week
	Interval kline.Interval
}

// FuturesMarketAnalyticsRequest holds the parameters of Market Analytics
type FuturesMarketAnalyticsRequest struct {
	Pair currency.Pair
	// AnalyticsType is open-interest, aggressor-differential, trade-volume, trade-count, liquidation-volume,
	// rolling-volatility, long-short-ratio, long-short-info, cvd, top-traders, orderbook, spreads, liquidity, slippage,
	// future-basis or funding
	AnalyticsType string
	// Since is the start of the first bucket, though the rolling-volatility type also returns the bucket before it
	Since time.Time
	// To returns the buckets up to it rather than up to now
	To time.Time
	// Interval is the bucket size: 1, 5, 15 or 30 minutes, 1, 4 or 12 hours, 1 day or 1 week
	Interval kline.Interval
}

// futuresAnalyticsEnvelope is the body of the analytics endpoints, which report problems in errors rather than in the
// fields of the Derivatives REST v3 API
type futuresAnalyticsEnvelope struct {
	Result *FuturesAnalyticsResponse `json:"result"`
	Errors []futuresAnalyticsError   `json:"errors"`
}

// futuresAnalyticsError is a problem an analytics endpoint reports, such as an invalid argument and the field it is in
type futuresAnalyticsError struct {
	// Severity is E for an error, or W for a warning as in Spot REST messages
	Severity   string `json:"severity"`
	ErrorClass string `json:"error_class"`
	Type       string `json:"type"`
	Message    string `json:"msg"`
	Value      string `json:"value"`
	Field      string `json:"field"`
}

// message formats the problem as a Spot REST error, such as EGeneral:Invalid arguments:interval
func (f *futuresAnalyticsError) message() string {
	parts := []string{f.Severity + f.ErrorClass, f.Type}
	for _, detail := range []string{f.Field, f.Message, f.Value} {
		if detail != "" {
			parts = append(parts, detail)
		}
	}
	return strings.Join(parts, ":")
}

// FuturesAnalyticsResponse holds a statistic in time buckets
type FuturesAnalyticsResponse struct {
	// Timestamps are the buckets' start times, each matching the values at its index in Data's series. The funding
	// type sends them in milliseconds, the others in seconds
	Timestamps []types.Time         `json:"timestamp"`
	Data       FuturesAnalyticsData `json:"data"`
	// More is set when the range holds more buckets after the last one returned
	More bool `json:"more"`
}

// FuturesAnalyticsData holds the series of a statistic, each holding a value per bucket. Only the series of the
// requested statistic are set
type FuturesAnalyticsData struct {
	// Values is the series of the aggressor-differential, trade-volume, trade-count, liquidation-volume,
	// rolling-volatility and long-short-ratio types
	Values []types.Number
	// OHLC is the series of the open-interest type
	OHLC []FuturesAnalyticsOHLC
	// LongCount, ShortCount, LongPercent, ShortPercent and Ratio are the series of the long-short-info type
	LongCount    []uint64
	ShortCount   []uint64
	LongPercent  []types.Number
	ShortPercent []types.Number
	Ratio        []types.Number
	// Bid and Ask hold the series of the orderbook, spreads, liquidity and slippage types
	Bid FuturesAnalyticsBookSide
	Ask FuturesAnalyticsBookSide
	// BuyVolume, SellVolume and CumulativeVolumeDelta are the series of the cvd type
	BuyVolume             []types.Number
	SellVolume            []types.Number
	CumulativeVolumeDelta []types.Number
	// Top20Percent holds the series of the top-traders type
	Top20Percent FuturesAnalyticsTopTraders
	// USDValue is the series of the liquidity pool statistic
	USDValue []types.Number
	// Basis is the series of the future-basis type
	Basis []types.Number
	// FundingRate and RelativeFundingRate are the series of the funding type
	FundingRate         []FuturesAnalyticsOHLC
	RelativeFundingRate []FuturesAnalyticsOHLC
}

// UnmarshalJSON decodes the series of any statistic: an array of values or of [open, high, low, close] arrays, or an
// object of named series. Kraken documents the cvd type's buy and sell volumes as buyVolume and sellVolume but sends
// buy_volume and sell_volume, so both are accepted
func (f *FuturesAnalyticsData) UnmarshalJSON(data []byte) error {
	if len(data) != 0 && data[0] == '[' {
		var elements []json.RawMessage
		if err := json.Unmarshal(data, &elements); err != nil {
			return err
		}
		if len(elements) != 0 && len(elements[0]) != 0 && elements[0][0] == '[' {
			return json.Unmarshal(data, &f.OHLC)
		}
		return json.Unmarshal(data, &f.Values)
	}
	var series map[string]json.RawMessage
	if err := json.Unmarshal(data, &series); err != nil {
		return err
	}
	for name, values := range series {
		var target any
		switch name {
		case "longCount":
			target = &f.LongCount
		case "shortCount":
			target = &f.ShortCount
		case "longPercent":
			target = &f.LongPercent
		case "shortPercent":
			target = &f.ShortPercent
		case "ratio":
			target = &f.Ratio
		case "bid":
			target = &f.Bid
		case "ask":
			target = &f.Ask
		case "buyVolume", "buy_volume":
			target = &f.BuyVolume
		case "sellVolume", "sell_volume":
			target = &f.SellVolume
		case "cvd":
			target = &f.CumulativeVolumeDelta
		case "top20Percent":
			target = &f.Top20Percent
		case "usdValue":
			target = &f.USDValue
		case "basis":
			target = &f.Basis
		case "rate":
			target = &f.FundingRate
		case "relativeRate":
			target = &f.RelativeFundingRate
		default:
			continue
		}
		if err := json.Unmarshal(values, target); err != nil {
			return err
		}
	}
	return nil
}

// FuturesAnalyticsOHLC holds a bucket's open, high, low and close values
type FuturesAnalyticsOHLC struct {
	Open  types.Number
	High  types.Number
	Low   types.Number
	Close types.Number
}

// UnmarshalJSON decodes an [open, high, low, close] array
func (f *FuturesAnalyticsOHLC) UnmarshalJSON(data []byte) error {
	fields := [4]any{&f.Open, &f.High, &f.Low, &f.Close}
	return unmarshalFixedArray(data, fields[:])
}

// FuturesAnalyticsBookSide holds the series of a side of the order book. The spreads, liquidity and slippage types
// each send their own series, in snake case, and the orderbook type sends them all, in camel case. A null value, such
// as the slippage of an order the book is too thin to fill, decodes as 0
type FuturesAnalyticsBookSide struct {
	BestPrice []types.Number
	// Liquidity005 to Liquidity100 are the liquidity within bands Kraken does not document; comparing them with the
	// book suggests the volume within 0.5, 1, 2.5, 5, 10 and 100 percent of the best price
	Liquidity005 []types.Number
	Liquidity01  []types.Number
	Liquidity025 []types.Number
	Liquidity05  []types.Number
	Liquidity10  []types.Number
	Liquidity100 []types.Number
	// Slippage1Thousand to Slippage1Million appear to be the average prices that orders of 1 thousand to 1 million in
	// notional value would fill at
	Slippage1Thousand   []types.Number
	Slippage10Thousand  []types.Number
	Slippage100Thousand []types.Number
	Slippage1Million    []types.Number
}

// UnmarshalJSON decodes the series of either case
func (f *FuturesAnalyticsBookSide) UnmarshalJSON(data []byte) error {
	var series map[string][]types.Number
	if err := json.Unmarshal(data, &series); err != nil {
		return err
	}
	for name, values := range series {
		switch name {
		case "best_price", "bestPrice":
			f.BestPrice = values
		case "liquidity_005", "liquidity005":
			f.Liquidity005 = values
		case "liquidity_01", "liquidity01":
			f.Liquidity01 = values
		case "liquidity_025", "liquidity025":
			f.Liquidity025 = values
		case "liquidity_05", "liquidity05":
			f.Liquidity05 = values
		case "liquidity_10", "liquidity10":
			f.Liquidity10 = values
		case "liquidity_100", "liquidity100":
			f.Liquidity100 = values
		case "slippage_1k", "slippage1k":
			f.Slippage1Thousand = values
		case "slippage_10k", "slippage10k":
			f.Slippage10Thousand = values
		case "slippage_100k", "slippage100k":
			f.Slippage100Thousand = values
		case "slippage_1m", "slippage1m":
			f.Slippage1Million = values
		}
	}
	return nil
}

// FuturesAnalyticsTopTraders holds the series of the traders with the largest positions
type FuturesAnalyticsTopTraders struct {
	OpenInterest []types.Number `json:"openInterest"`
	LongCount    []uint64       `json:"longCount"`
	ShortCount   []uint64       `json:"shortCount"`
	LongPercent  []types.Number `json:"longPercent"`
	ShortPercent []types.Number `json:"shortPercent"`
	Ratio        []types.Number `json:"ratio"`
}
