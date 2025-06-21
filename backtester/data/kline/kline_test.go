package kline

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	p := currency.NewBTCUSDT()
	tt := time.Now()
	d := DataFromKline{
		Base: &data.Base{},
	}
	err := d.Load()
	assert.ErrorIs(t, err, errNoCandleData)

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
	assert.NoError(t, err)
}

func TestHasDataAtTime(t *testing.T) {
	t.Parallel()
	dStart := time.Date(2020, 1, 0, 0, 0, 0, 0, time.UTC)
	dEnd := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	exch := testExchange
	a := asset.Spot
	p := currency.NewBTCUSDT()
	d := DataFromKline{
		Base: &data.Base{},
	}
	has, err := d.HasDataAtTime(time.Now())
	require.ErrorIs(t, err, gctcommon.ErrNilPointer)
	assert.False(t, has)

	d.RangeHolder = &gctkline.IntervalRangeHolder{}
	has, err = d.HasDataAtTime(time.Now())
	require.NoError(t, err)
	assert.False(t, has)

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
	require.NoError(t, d.Load(), "Load must not error")
	has, err = d.HasDataAtTime(dStart)
	require.NoError(t, err)
	assert.False(t, has)

	ranger, err := gctkline.CalculateCandleDateRanges(dStart, dEnd, gctkline.OneDay, 100000)
	require.NoError(t, err)

	d.RangeHolder = ranger
	err = d.RangeHolder.SetHasDataFromCandles(d.Item.Candles)
	require.NoError(t, err)

	has, err = d.HasDataAtTime(dStart)
	require.NoError(t, err)
	assert.True(t, has)

	err = d.SetLive(true)
	require.NoError(t, err)

	has, err = d.HasDataAtTime(time.Time{})
	require.NoError(t, err)
	assert.False(t, has)

	has, err = d.HasDataAtTime(dStart)
	require.NoError(t, err)
	assert.True(t, has)
}

func TestAppend(t *testing.T) {
	t.Parallel()
	a := asset.Spot
	p := currency.NewBTCUSDT()
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
	assert.ErrorIs(t, err, gctkline.ErrItemNotEqual)

	item.Exchange = testExchange
	item.Pair = p
	item.Asset = a

	err = d.AppendResults(&item)
	assert.NoError(t, err)

	err = d.AppendResults(&item)
	assert.NoError(t, err)

	err = d.AppendResults(nil)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)
}

func TestStreamOpen(t *testing.T) {
	t.Parallel()
	exch := testExchange
	a := asset.Spot
	p := currency.NewBTCUSDT()
	d := DataFromKline{
		Base: &data.Base{},
	}
	bad, err := d.StreamOpen()
	require.NoError(t, err)
	assert.Empty(t, bad, "StreamOpen should return an empty slice when no data is set")

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
	require.NoError(t, err)

	_, err = d.Next()
	require.NoError(t, err)

	open, err := d.StreamOpen()
	require.NoError(t, err)
	assert.NotEmpty(t, open, "open should not be empty")
}

func TestStreamVolume(t *testing.T) {
	t.Parallel()
	exch := testExchange
	a := asset.Spot
	p := currency.NewBTCUSDT()
	d := DataFromKline{
		Base: &data.Base{},
	}
	bad, err := d.StreamVol()
	assert.NoError(t, err)

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
	assert.NoError(t, err)

	_, err = d.Next()
	assert.NoError(t, err)

	vol, err := d.StreamVol()
	require.NoError(t, err)
	assert.NotEmpty(t, vol, "StreamVol should return a non-empty slice")
}

func TestStreamClose(t *testing.T) {
	t.Parallel()
	exch := testExchange
	a := asset.Spot
	p := currency.NewBTCUSDT()
	d := DataFromKline{
		Base: &data.Base{},
	}
	bad, err := d.StreamClose()
	assert.NoError(t, err)

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
	assert.NoError(t, err)

	_, err = d.Next()
	assert.NoError(t, err)

	cl, err := d.StreamClose()
	require.NoError(t, err)
	assert.NotEmpty(t, cl, "StreamClose should return a non-empty slice")
}

func TestStreamHigh(t *testing.T) {
	t.Parallel()
	exch := testExchange
	a := asset.Spot
	p := currency.NewBTCUSDT()
	d := DataFromKline{
		Base: &data.Base{},
	}
	bad, err := d.StreamHigh()
	assert.NoError(t, err)

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
	assert.NoError(t, err)

	_, err = d.Next()
	assert.NoError(t, err)

	high, err := d.StreamHigh()
	assert.NoError(t, err)

	if len(high) == 0 {
		t.Error("expected high")
	}
}

func TestStreamLow(t *testing.T) {
	t.Parallel()
	exch := testExchange
	a := asset.Spot
	p := currency.NewBTCUSDT()
	d := DataFromKline{
		Base:        &data.Base{},
		RangeHolder: &gctkline.IntervalRangeHolder{},
	}
	bad, err := d.StreamLow()
	assert.NoError(t, err)

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
	assert.NoError(t, err)

	_, err = d.Next()
	assert.NoError(t, err)

	low, err := d.StreamLow()
	assert.NoError(t, err)

	if len(low) == 0 {
		t.Error("expected low")
	}
}
