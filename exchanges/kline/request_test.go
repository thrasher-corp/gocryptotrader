package kline

import (
	"errors"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

func TestCreateKlineRequest(t *testing.T) {
	t.Parallel()
	_, err := CreateKlineRequest("", currency.EMPTYPAIR, currency.EMPTYPAIR, 0, 0, 0, time.Time{}, time.Time{})
	if !errors.Is(err, ErrUnsetName) {
		t.Fatalf("received: '%v', but expected '%v'", err, ErrUnsetName)
	}

	_, err = CreateKlineRequest("name", currency.EMPTYPAIR, currency.EMPTYPAIR, 0, 0, 0, time.Time{}, time.Time{})
	if !errors.Is(err, currency.ErrCurrencyPairEmpty) {
		t.Fatalf("received: '%v', but expected '%v'", err, currency.ErrCurrencyPairEmpty)
	}

	pair := currency.NewPair(currency.BTC, currency.USDT)
	_, err = CreateKlineRequest("name", pair, currency.EMPTYPAIR, 0, 0, 0, time.Time{}, time.Time{})
	if !errors.Is(err, currency.ErrCurrencyPairEmpty) {
		t.Fatalf("received: '%v', but expected '%v'", err, currency.ErrCurrencyPairEmpty)
	}

	pair2 := pair.Upper()
	_, err = CreateKlineRequest("name", pair, pair2, 0, 0, 0, time.Time{}, time.Time{})
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("received: '%v', but expected '%v'", err, asset.ErrNotSupported)
	}

	_, err = CreateKlineRequest("name", pair, pair2, asset.Spot, 0, 0, time.Time{}, time.Time{})
	if !errors.Is(err, ErrUnsetInterval) {
		t.Fatalf("received: '%v', but expected '%v'", err, ErrUnsetInterval)
	}

	_, err = CreateKlineRequest("name", pair, pair2, asset.Spot, OneHour, 0, time.Time{}, time.Time{})
	if !errors.Is(err, ErrUnsetInterval) {
		t.Fatalf("received: '%v', but expected '%v'", err, ErrUnsetInterval)
	}

	_, err = CreateKlineRequest("name", pair, pair2, asset.Spot, OneHour, OneMin, time.Time{}, time.Time{})
	if !errors.Is(err, common.ErrDateUnset) {
		t.Fatalf("received: '%v', but expected '%v'", err, common.ErrDateUnset)
	}

	start := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	_, err = CreateKlineRequest("name", pair, pair2, asset.Spot, OneHour, OneMin, start, time.Time{})
	if !errors.Is(err, common.ErrDateUnset) {
		t.Fatalf("received: '%v', but expected '%v'", err, common.ErrDateUnset)
	}

	end := start.AddDate(0, 0, 1)
	r, err := CreateKlineRequest("name", pair, pair2, asset.Spot, OneHour, OneMin, start, end)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v', but expected '%v'", err, nil)
	}

	if r.Name != "name" {
		t.Fatalf("received: '%v' but expected: '%v'", r.Name, "name")
	}

	if !r.Pair.Equal(pair) {
		t.Fatalf("received: '%v' but expected: '%v'", r.Pair, pair)
	}

	if r.Asset != asset.Spot {
		t.Fatalf("received: '%v' but expected: '%v'", r.Asset, asset.Spot)
	}

	if r.Outbound != OneMin {
		t.Fatalf("received: '%v' but expected: '%v'", r.Outbound, OneMin)
	}

	if r.Required != OneHour {
		t.Fatalf("received: '%v' but expected: '%v'", r.Required, OneHour)
	}

	if r.Start != start {
		t.Fatalf("received: '%v' but expected: '%v'", r.Start, start)
	}

	if r.End != end {
		t.Fatalf("received: '%v' but expected: '%v'", r.End, end)
	}

	if r.Formatted.String() != "BTCUSDT" {
		t.Fatalf("received: '%v' but expected: '%v'", r.Formatted.String(), "BTCUSDT")
	}
}

func TestGetRanges(t *testing.T) {
	t.Parallel()

	start := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 1)
	pair := currency.NewPair(currency.BTC, currency.USDT)

	var r *Request
	_, err := r.GetRanges(100)
	if !errors.Is(err, errNilRequest) {
		t.Fatalf("received: '%v', but expected '%v'", err, errNilRequest)
	}

	r, err = CreateKlineRequest("name", pair, pair, asset.Spot, OneHour, OneMin, start, end)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v', but expected '%v'", err, nil)
	}

	holder, err := r.GetRanges(100)
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

func TestRequest_ConvertCandles(t *testing.T) {
	t.Parallel()

	start := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 1)
	pair := currency.NewPair(currency.BTC, currency.USDT)

	var r *Request
	_, err := r.ConvertCandles(oneHourCandles)
	if !errors.Is(err, errNilRequest) {
		t.Fatalf("received: '%v', but expected '%v'", err, errNilRequest)
	}

	// no conversion
	r, err = CreateKlineRequest("name", pair, pair, asset.Spot, OneHour, OneHour, start, end)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v', but expected '%v'", err, nil)
	}

	holder, err := r.ConvertCandles(oneHourCandles)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v', but expected '%v'", err, nil)
	}

	if len(holder.Candles) != 24 {
		t.Fatalf("received: '%v', but expected '%v'", len(holder.Candles), 24)
	}

	// with conversion
	r, err = CreateKlineRequest("name", pair, pair, asset.Spot, OneHour, OneMin, start, end)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v', but expected '%v'", err, nil)
	}

	holder, err = r.ConvertCandles(oneMinuteCandles)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v', but expected '%v'", err, nil)
	}

	if len(holder.Candles) != 24 {
		t.Fatalf("received: '%v', but expected '%v'", len(holder.Candles), 24)
	}
}

func TestRequestExtended_ConvertCandles(t *testing.T) {
	t.Parallel()

	start := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 1)
	pair := currency.NewPair(currency.BTC, currency.USDT)

	var rExt *RequestExtended
	_, err := rExt.ConvertCandles(oneHourCandles)
	if !errors.Is(err, errNilRequest) {
		t.Fatalf("received: '%v', but expected '%v'", err, errNilRequest)
	}

	// no conversion
	r, err := CreateKlineRequest("name", pair, pair, asset.Spot, OneHour, OneHour, start, end)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v', but expected '%v'", err, nil)
	}

	dates, err := r.GetRanges(100)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v', but expected '%v'", err, nil)
	}

	rExt = &RequestExtended{r, dates}

	holder, err := rExt.ConvertCandles(oneHourCandles)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v', but expected '%v'", err, nil)
	}

	if len(holder.Candles) != 24 {
		t.Fatalf("received: '%v', but expected '%v'", len(holder.Candles), 24)
	}

	// with conversion
	r, err = CreateKlineRequest("name", pair, pair, asset.Spot, OneHour, OneMin, start, end)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v', but expected '%v'", err, nil)
	}

	dates, err = r.GetRanges(100)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v', but expected '%v'", err, nil)
	}

	rExt = &RequestExtended{r, dates}

	holder, err = rExt.ConvertCandles(oneMinuteCandles)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v', but expected '%v'", err, nil)
	}

	if len(holder.Candles) != 24 {
		t.Fatalf("received: '%v', but expected '%v'", len(holder.Candles), 24)
	}
}

func TestRequestExtended_Size(t *testing.T) {
	t.Parallel()

	var rExt *RequestExtended
	if rExt.Size() != 0 {
		t.Fatalf("received: '%v', but expected '%v'", rExt.Size(), 0)
	}

	rExt = &RequestExtended{IntervalRangeHolder: &IntervalRangeHolder{Limit: 100, Ranges: []IntervalRange{{}, {}}}}
	if rExt.Size() != 200 {
		t.Fatalf("received: '%v', but expected '%v'", rExt.Size(), 200)
	}
}
