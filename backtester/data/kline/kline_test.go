package kline

import (
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/kline"
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
	d := DataFromKline{}
	err := d.Load()
	if !errors.Is(err, errNoCandleData) {
		t.Errorf("received: %v, expected: %v", err, errNoCandleData)
	}
	d.Item = gctkline.Item{
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
	dInsert := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	dEnd := time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC)
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	d := DataFromKline{}
	has := d.HasDataAtTime(time.Now())
	if has {
		t.Error("expected false")
	}

	d.Item = gctkline.Item{
		Exchange: exch,
		Pair:     p,
		Asset:    a,
		Interval: gctkline.OneDay,
		Candles: []gctkline.Candle{
			{
				Time:   dInsert,
				Open:   1337,
				High:   1337,
				Low:    1337,
				Close:  1337,
				Volume: 1337,
			},
		},
	}
	if err := d.Load(); err != nil {
		t.Error(err)
	}

	has = d.HasDataAtTime(dInsert)
	if has {
		t.Error("expected false")
	}

	ranger, err := gctkline.CalculateCandleDateRanges(dStart, dEnd, gctkline.OneDay, 100000)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	d.RangeHolder = ranger
	d.RangeHolder.SetHasDataFromCandles(d.Item.Candles)
	has = d.HasDataAtTime(dInsert)
	if !has {
		t.Error("expected true")
	}
	d.SetLive(true)
	has = d.HasDataAtTime(time.Time{})
	if has {
		t.Error("expected false")
	}
	has = d.HasDataAtTime(dInsert)
	if !has {
		t.Error("expected true")
	}
}

func TestAppend(t *testing.T) {
	t.Parallel()
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	d := DataFromKline{
		Item: gctkline.Item{
			Exchange: testExchange,
			Asset:    a,
			Pair:     p,
		},
		RangeHolder: &gctkline.IntervalRangeHolder{},
	}
	item := gctkline.Item{
		Interval: gctkline.OneDay,
		Candles: []gctkline.Candle{
			{
				Time:   time.Now(),
				Open:   1337,
				High:   1337,
				Low:    1337,
				Close:  1337,
				Volume: 1337,
			},
		},
	}
	d.AppendResults(&item)

	item.Exchange = testExchange
	item.Pair = p
	item.Asset = a
	d.AppendResults(&item)
	d.AppendResults(&item)
	d.AppendResults(nil)
}

func TestStreamOpen(t *testing.T) {
	t.Parallel()
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	d := DataFromKline{}
	if bad := d.StreamOpen(); len(bad) > 0 {
		t.Error("expected no stream")
	}
	d.SetStream([]data.Event{
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
	d.Next()
	if open := d.StreamOpen(); len(open) == 0 {
		t.Error("expected open")
	}
}

func TestStreamVolume(t *testing.T) {
	t.Parallel()
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	d := DataFromKline{}
	if bad := d.StreamVol(); len(bad) > 0 {
		t.Error("expected no stream")
	}
	d.SetStream([]data.Event{
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
	d.Next()
	if open := d.StreamVol(); len(open) == 0 {
		t.Error("expected volume")
	}
}

func TestStreamClose(t *testing.T) {
	t.Parallel()
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	d := DataFromKline{}
	if bad := d.StreamClose(); len(bad) > 0 {
		t.Error("expected no stream")
	}
	d.SetStream([]data.Event{
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
	d.Next()
	if open := d.StreamClose(); len(open) == 0 {
		t.Error("expected close")
	}
}

func TestStreamHigh(t *testing.T) {
	t.Parallel()
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	d := DataFromKline{}
	if bad := d.StreamHigh(); len(bad) > 0 {
		t.Error("expected no stream")
	}
	d.SetStream([]data.Event{
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
	d.Next()
	if open := d.StreamHigh(); len(open) == 0 {
		t.Error("expected high")
	}
}

func TestStreamLow(t *testing.T) {
	t.Parallel()
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	d := DataFromKline{
		RangeHolder: &gctkline.IntervalRangeHolder{},
	}
	if bad := d.StreamLow(); len(bad) > 0 {
		t.Error("expected no stream")
	}
	d.SetStream([]data.Event{
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
	d.Next()
	if open := d.StreamLow(); len(open) == 0 {
		t.Error("expected low")
	}
}
