package kline

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

func TestGetBuilder(t *testing.T) {
	t.Parallel()
	_, err := GetBuilder("", currency.EMPTYPAIR, currency.EMPTYPAIR, 0, 0, 0, time.Time{}, time.Time{})
	if !errors.Is(err, ErrUnsetName) {
		t.Fatalf("received: '%v', but expected '%v'", err, ErrUnsetName)
	}

	_, err = GetBuilder("name", currency.EMPTYPAIR, currency.EMPTYPAIR, 0, 0, 0, time.Time{}, time.Time{})
	if !errors.Is(err, currency.ErrCurrencyPairEmpty) {
		t.Fatalf("received: '%v', but expected '%v'", err, currency.ErrCurrencyPairEmpty)
	}

	pair := currency.NewPair(currency.BTC, currency.USDT)
	_, err = GetBuilder("name", pair, currency.EMPTYPAIR, 0, 0, 0, time.Time{}, time.Time{})
	if !errors.Is(err, currency.ErrCurrencyPairEmpty) {
		t.Fatalf("received: '%v', but expected '%v'", err, currency.ErrCurrencyPairEmpty)
	}

	pair2 := pair.Upper()
	_, err = GetBuilder("name", pair, pair2, 0, 0, 0, time.Time{}, time.Time{})
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("received: '%v', but expected '%v'", err, asset.ErrNotSupported)
	}

	_, err = GetBuilder("name", pair, pair2, asset.Spot, 0, 0, time.Time{}, time.Time{})
	if !errors.Is(err, ErrUnsetInterval) {
		t.Fatalf("received: '%v', but expected '%v'", err, ErrUnsetInterval)
	}

	_, err = GetBuilder("name", pair, pair2, asset.Spot, OneHour, 0, time.Time{}, time.Time{})
	if !errors.Is(err, ErrUnsetInterval) {
		t.Fatalf("received: '%v', but expected '%v'", err, ErrUnsetInterval)
	}

	_, err = GetBuilder("name", pair, pair2, asset.Spot, OneHour, OneMin, time.Time{}, time.Time{})
	if !errors.Is(err, common.ErrDateUnset) {
		t.Fatalf("received: '%v', but expected '%v'", err, common.ErrDateUnset)
	}

	start := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	_, err = GetBuilder("name", pair, pair2, asset.Spot, OneHour, OneMin, start, time.Time{})
	if !errors.Is(err, common.ErrDateUnset) {
		t.Fatalf("received: '%v', but expected '%v'", err, common.ErrDateUnset)
	}

	end := start.AddDate(0, 0, 1)
	builder, err := GetBuilder("name", pair, pair2, asset.Spot, OneHour, OneMin, start, end)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v', but expected '%v'", err, nil)
	}

	if builder.Name != "name" {
		t.Fatalf("received: '%v' but expected: '%v'", builder.Name, "name")
	}

	if !builder.Pair.Equal(pair) {
		t.Fatalf("received: '%v' but expected: '%v'", builder.Pair, pair)
	}

	if builder.Asset != asset.Spot {
		t.Fatalf("received: '%v' but expected: '%v'", builder.Asset, asset.Spot)
	}

	if builder.Request != OneMin {
		t.Fatalf("received: '%v' but expected: '%v'", builder.Request, OneMin)
	}

	if builder.Required != OneHour {
		t.Fatalf("received: '%v' but expected: '%v'", builder.Required, OneHour)
	}

	if builder.Start != start {
		t.Fatalf("received: '%v' but expected: '%v'", builder.Start, start)
	}

	if builder.End != end {
		t.Fatalf("received: '%v' but expected: '%v'", builder.End, end)
	}

	if builder.Formatted.String() != "BTCUSDT" {
		t.Fatalf("received: '%v' but expected: '%v'", builder.Formatted.String(), "BTCUSDT")
	}
}

func TestGetRanges(t *testing.T) {
	t.Parallel()

	start := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 1)
	pair := currency.NewPair(currency.BTC, currency.USDT)
	builder, err := GetBuilder("name", pair, pair, asset.Spot, OneHour, OneMin, start, end)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v', but expected '%v'", err, nil)
	}

	holder, err := builder.GetRanges(100)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v', but expected '%v'", err, nil)
	}

	if len(holder.Ranges) != 15 {
		t.Fatalf("received: '%v', but expected '%v'", len(holder.Ranges), 15)
	}
}

var oneMinuteCandles = func() []Candle {
	var candles []Candle
	start := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	for x := 0; x < 1442; x++ { // two extra candles.
		candles = append(candles, Candle{
			Time:   start,
			Volume: 1,
			Open:   1,
			High:   float64(1 + x),
			Low:    float64(-(1 + x)),
			Close:  1,
		})
		start = start.Add(time.Minute)
	}
	return candles
}()

var oneHourCandles = func() []Candle {
	var candles []Candle
	start := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	for x := 0; x < 24; x++ {
		candles = append(candles, Candle{
			Time:   start,
			Volume: 1,
			Open:   1,
			High:   float64(1 + x),
			Low:    float64(-(1 + x)),
			Close:  1,
		})
		start = start.Add(time.Hour)
	}
	return candles
}()

func TestBuilder_ConvertCandles(t *testing.T) {
	t.Parallel()

	start := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 1)
	pair := currency.NewPair(currency.BTC, currency.USDT)

	// no conversion
	builder, err := GetBuilder("name", pair, pair, asset.Spot, OneHour, OneHour, start, end)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v', but expected '%v'", err, nil)
	}

	holder, err := builder.ConvertCandles(oneHourCandles)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v', but expected '%v'", err, nil)
	}

	if len(holder.Candles) != 24 {
		t.Fatalf("received: '%v', but expected '%v'", len(holder.Candles), 24)
	}

	// with conversion
	builder, err = GetBuilder("name", pair, pair, asset.Spot, OneHour, OneMin, start, end)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v', but expected '%v'", err, nil)
	}

	holder, err = builder.ConvertCandles(oneMinuteCandles)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v', but expected '%v'", err, nil)
	}

	fmt.Printf("moo: '%+v'\n", holder)
	fmt.Printf("moo: '%+v'\n", len(holder.Candles))

	if len(holder.Candles) != 24 {
		t.Fatalf("received: '%v', but expected '%v'", len(holder.Candles), 24)
	}
}

func TestBuilderExtended_ConvertCandles(t *testing.T) {
	t.Parallel()

	start := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 1)
	pair := currency.NewPair(currency.BTC, currency.USDT)

	// no conversion
	builder, err := GetBuilder("name", pair, pair, asset.Spot, OneHour, OneHour, start, end)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v', but expected '%v'", err, nil)
	}

	dates, err := builder.GetRanges(100)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v', but expected '%v'", err, nil)
	}

	buildExt := BuilderExtended{builder, dates}

	holder, err := buildExt.ConvertCandles(oneHourCandles)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v', but expected '%v'", err, nil)
	}

	if len(holder.Candles) != 24 {
		t.Fatalf("received: '%v', but expected '%v'", len(holder.Candles), 24)
	}

	// with conversion
	builder, err = GetBuilder("name", pair, pair, asset.Spot, OneHour, OneMin, start, end)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v', but expected '%v'", err, nil)
	}

	dates, err = builder.GetRanges(100)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v', but expected '%v'", err, nil)
	}

	buildExt = BuilderExtended{builder, dates}

	holder, err = buildExt.ConvertCandles(oneMinuteCandles)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v', but expected '%v'", err, nil)
	}

	if len(holder.Candles) != 24 {
		t.Fatalf("received: '%v', but expected '%v'", len(holder.Candles), 24)
	}
}
