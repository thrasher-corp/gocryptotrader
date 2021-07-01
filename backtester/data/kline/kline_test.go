package kline

import (
	"errors"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/kline"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

const testExchange = "binance"

func TestLoad(t *testing.T) {
	t.Parallel()
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	tt := time.Now()
	d := DataFromKline{}
	err := d.Load()
	if !errors.Is(err, errNoCandleData) {
		t.Errorf("expected: %v, received %v", errNoCandleData, err)
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
	if err != nil {
		t.Error(err)
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
	err := d.Load()
	if err != nil {
		t.Error(err)
	}

	has = d.HasDataAtTime(dInsert)
	if has {
		t.Error("expected false")
	}

	ranger, err := gctkline.CalculateCandleDateRanges(dStart, dEnd, gctkline.OneDay, 100000)
	if err != nil {
		t.Error(err)
	}
	d.Range = ranger
	d.Range.SetHasDataFromCandles(d.Item.Candles)
	has = d.HasDataAtTime(dInsert)
	if !has {
		t.Error("expected true")
	}
}

func TestAppend(t *testing.T) {
	t.Parallel()
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	d := DataFromKline{}
	item := gctkline.Item{
		Exchange: exch,
		Pair:     p,
		Asset:    a,
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
	d.Append(&item)
}

func TestStreamOpen(t *testing.T) {
	t.Parallel()
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	d := DataFromKline{}
	bad := d.StreamOpen()
	if len(bad) > 0 {
		t.Error("expected no stream")
	}
	d.SetStream([]common.DataEventHandler{
		&kline.Kline{
			Base: event.Base{
				Exchange:     exch,
				Time:         time.Now(),
				Interval:     gctkline.OneDay,
				CurrencyPair: p,
				AssetType:    a,
			},
			Open:   1337,
			High:   1337,
			Low:    1337,
			Close:  1337,
			Volume: 1337,
		},
	})
	d.Next()
	open := d.StreamOpen()
	if len(open) == 0 {
		t.Error("expected open")
	}
}

func TestStreamVolume(t *testing.T) {
	t.Parallel()
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	d := DataFromKline{}
	bad := d.StreamVol()
	if len(bad) > 0 {
		t.Error("expected no stream")
	}
	d.SetStream([]common.DataEventHandler{
		&kline.Kline{
			Base: event.Base{
				Exchange:     exch,
				Time:         time.Now(),
				Interval:     gctkline.OneDay,
				CurrencyPair: p,
				AssetType:    a,
			},
			Open:   1337,
			High:   1337,
			Low:    1337,
			Close:  1337,
			Volume: 1337,
		},
	})
	d.Next()
	open := d.StreamVol()
	if len(open) == 0 {
		t.Error("expected volume")
	}
}

func TestStreamClose(t *testing.T) {
	t.Parallel()
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	d := DataFromKline{}
	bad := d.StreamClose()
	if len(bad) > 0 {
		t.Error("expected no stream")
	}
	d.SetStream([]common.DataEventHandler{
		&kline.Kline{
			Base: event.Base{
				Exchange:     exch,
				Time:         time.Now(),
				Interval:     gctkline.OneDay,
				CurrencyPair: p,
				AssetType:    a,
			},
			Open:   1337,
			High:   1337,
			Low:    1337,
			Close:  1337,
			Volume: 1337,
		},
	})
	d.Next()
	open := d.StreamClose()
	if len(open) == 0 {
		t.Error("expected close")
	}
}

func TestStreamHigh(t *testing.T) {
	t.Parallel()
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	d := DataFromKline{}
	bad := d.StreamHigh()
	if len(bad) > 0 {
		t.Error("expected no stream")
	}
	d.SetStream([]common.DataEventHandler{
		&kline.Kline{
			Base: event.Base{
				Exchange:     exch,
				Time:         time.Now(),
				Interval:     gctkline.OneDay,
				CurrencyPair: p,
				AssetType:    a,
			},
			Open:   1337,
			High:   1337,
			Low:    1337,
			Close:  1337,
			Volume: 1337,
		},
	})
	d.Next()
	open := d.StreamHigh()
	if len(open) == 0 {
		t.Error("expected high")
	}
}

func TestStreamLow(t *testing.T) {
	t.Parallel()
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	d := DataFromKline{}
	bad := d.StreamLow()
	if len(bad) > 0 {
		t.Error("expected no stream")
	}
	d.SetStream([]common.DataEventHandler{
		&kline.Kline{
			Base: event.Base{
				Exchange:     exch,
				Time:         time.Now(),
				Interval:     gctkline.OneDay,
				CurrencyPair: p,
				AssetType:    a,
			},
			Open:   1337,
			High:   1337,
			Low:    1337,
			Close:  1337,
			Volume: 1337,
		},
	})
	d.Next()
	open := d.StreamLow()
	if len(open) == 0 {
		t.Error("expected low")
	}
}
