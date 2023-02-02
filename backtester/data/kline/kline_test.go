package kline

import (
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/kline"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

const testExchange = "binance"

var elite = decimal.NewFromInt(1337)

func TestLoad(t *testing.T) {
	t.Parallel()
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	tt := time.Now()
	d := DataFromKline{
		Base: &data.Base{},
	}
	err := d.Load()
	if !errors.Is(err, errNoCandleData) {
		t.Errorf("received: %v, expected: %v", err, errNoCandleData)
	}
	d.Item = &gctkline.Item{
		Exchange: exch,
		Pair:     p,
		Asset:    a,
		Interval: gctkline.FifteenMin,
		Candles: []gctkline.Candle{
			{
				Time:   tt,
				Open:   1337,
				High:   1337,
				Low:    1337,
				Close:  1337,
				Volume: 1337,
			},
		},
	}
	err = d.Load()
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
}

func TestHasDataAtTime(t *testing.T) {
	t.Parallel()
	dStart := time.Date(2020, 1, 0, 0, 0, 0, 0, time.UTC)
	dEnd := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	d := DataFromKline{
		Base: &data.Base{},
	}
	has, err := d.HasDataAtTime(time.Now())
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received: %v, expected: %v", err, gctcommon.ErrNilPointer)
	}
	if has {
		t.Error("expected false")
	}

	d.RangeHolder = &gctkline.IntervalRangeHolder{}
	has, err = d.HasDataAtTime(time.Now())
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	if has {
		t.Error("expected false")
	}

	d.Item = &gctkline.Item{
		Exchange: exch,
		Pair:     p,
		Asset:    a,
		Interval: gctkline.OneDay,
		Candles: []gctkline.Candle{
			{
				Time:   dStart,
				Open:   1337,
				High:   1337,
				Low:    1337,
				Close:  1337,
				Volume: 1337,
			},
		},
	}
	if err = d.Load(); err != nil {
		t.Error(err)
	}

	has, err = d.HasDataAtTime(dStart)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	if has {
		t.Error("expected false")
	}

	ranger, err := gctkline.CalculateCandleDateRanges(dStart, dEnd, gctkline.OneDay, 100000)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	d.RangeHolder = ranger
	err = d.RangeHolder.SetHasDataFromCandles(d.Item.Candles)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	has, err = d.HasDataAtTime(dStart)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	if !has {
		t.Error("expected true")
	}
	err = d.SetLive(true)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	has, err = d.HasDataAtTime(time.Time{})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	if has {
		t.Error("expected false")
	}
	has, err = d.HasDataAtTime(dStart)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	if !has {
		t.Error("expected true")
	}
}

func TestAppend(t *testing.T) {
	t.Parallel()
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	tt1 := time.Date(2020, 1, 0, 0, 0, 0, 0, time.UTC)
	tt2 := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	d := DataFromKline{
		Base: &data.Base{},
		Item: &gctkline.Item{
			Exchange: testExchange,
			Asset:    a,
			Pair:     p,
			Interval: gctkline.OneDay,
		},
		RangeHolder: &gctkline.IntervalRangeHolder{},
	}
	item := gctkline.Item{
		Interval: gctkline.OneDay,
		Candles: []gctkline.Candle{
			{
				Time:   tt1,
				Open:   1337,
				High:   1337,
				Low:    1337,
				Close:  1337,
				Volume: 1337,
			},
			{
				Time:   tt2,
				Open:   1337,
				High:   1337,
				Low:    1337,
				Close:  1337,
				Volume: 1337,
			},
		},
	}
	err := d.AppendResults(&item)
	if !errors.Is(err, gctkline.ErrItemNotEqual) {
		t.Errorf("received: %v, expected: %v", err, gctkline.ErrItemNotEqual)
	}

	item.Exchange = testExchange
	item.Pair = p
	item.Asset = a

	err = d.AppendResults(&item)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}

	err = d.AppendResults(&item)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}

	err = d.AppendResults(nil)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received: %v, expected: %v", err, gctcommon.ErrNilPointer)
	}
}

func TestStreamOpen(t *testing.T) {
	t.Parallel()
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	d := DataFromKline{
		Base: &data.Base{},
	}
	bad, err := d.StreamOpen()
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	if len(bad) > 0 {
		t.Error("expected no stream")
	}
	err = d.SetStream([]data.Event{
		&kline.Kline{
			Base: &event.Base{
				Exchange:     exch,
				Time:         time.Now(),
				Interval:     gctkline.OneDay,
				CurrencyPair: p,
				AssetType:    a,
			},
			Open:   elite,
			High:   elite,
			Low:    elite,
			Close:  elite,
			Volume: elite,
		},
	})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	_, err = d.Next()
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	open, err := d.StreamOpen()
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	if len(open) == 0 {
		t.Error("expected open")
	}
}

func TestStreamVolume(t *testing.T) {
	t.Parallel()
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	d := DataFromKline{
		Base: &data.Base{},
	}
	bad, err := d.StreamVol()
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	if len(bad) > 0 {
		t.Error("expected no stream")
	}
	err = d.SetStream([]data.Event{
		&kline.Kline{
			Base: &event.Base{
				Exchange:     exch,
				Time:         time.Now(),
				Interval:     gctkline.OneDay,
				CurrencyPair: p,
				AssetType:    a,
			},
			Open:   elite,
			High:   elite,
			Low:    elite,
			Close:  elite,
			Volume: elite,
		},
	})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	_, err = d.Next()
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	vol, err := d.StreamVol()
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	if len(vol) == 0 {
		t.Error("expected volume")
	}
}

func TestStreamClose(t *testing.T) {
	t.Parallel()
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	d := DataFromKline{
		Base: &data.Base{},
	}
	bad, err := d.StreamClose()
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	if len(bad) > 0 {
		t.Error("expected no stream")
	}

	err = d.SetStream([]data.Event{
		&kline.Kline{
			Base: &event.Base{
				Exchange:     exch,
				Time:         time.Now(),
				Interval:     gctkline.OneDay,
				CurrencyPair: p,
				AssetType:    a,
			},
			Open:   elite,
			High:   elite,
			Low:    elite,
			Close:  elite,
			Volume: elite,
		},
	})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	_, err = d.Next()
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	cl, err := d.StreamClose()
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	if len(cl) == 0 {
		t.Error("expected close")
	}
}

func TestStreamHigh(t *testing.T) {
	t.Parallel()
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	d := DataFromKline{
		Base: &data.Base{},
	}
	bad, err := d.StreamHigh()
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	if len(bad) > 0 {
		t.Error("expected no stream")
	}

	err = d.SetStream([]data.Event{
		&kline.Kline{
			Base: &event.Base{
				Exchange:     exch,
				Time:         time.Now(),
				Interval:     gctkline.OneDay,
				CurrencyPair: p,
				AssetType:    a,
			},
			Open:   elite,
			High:   elite,
			Low:    elite,
			Close:  elite,
			Volume: elite,
		},
	})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	_, err = d.Next()
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	high, err := d.StreamHigh()
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	if len(high) == 0 {
		t.Error("expected high")
	}
}

func TestStreamLow(t *testing.T) {
	t.Parallel()
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	d := DataFromKline{
		Base:        &data.Base{},
		RangeHolder: &gctkline.IntervalRangeHolder{},
	}
	bad, err := d.StreamLow()
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	if len(bad) > 0 {
		t.Error("expected no stream")
	}

	err = d.SetStream([]data.Event{
		&kline.Kline{
			Base: &event.Base{
				Exchange:     exch,
				Time:         time.Now(),
				Interval:     gctkline.OneDay,
				CurrencyPair: p,
				AssetType:    a,
			},
			Open:   elite,
			High:   elite,
			Low:    elite,
			Close:  elite,
			Volume: elite,
		},
	})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	_, err = d.Next()
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}

	low, err := d.StreamLow()
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	if len(low) == 0 {
		t.Error("expected low")
	}
}
