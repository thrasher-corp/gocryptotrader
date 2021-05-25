package kline

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/kline"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// HasDataAtTime verifies checks the underlying range data
// To determine whether there is any candle data present at the time provided
func (d *DataFromKline) HasDataAtTime(t time.Time) bool {
	if d.Range == nil {
		return false
	}
	return d.Range.HasDataAtDate(t)
}

// Load sets the candle data to the stream for processing
func (d *DataFromKline) Load() error {
	d.addedTimes = make(map[time.Time]bool)
	if len(d.Item.Candles) == 0 {
		return errNoCandleData
	}

	klineData := make([]common.DataEventHandler, len(d.Item.Candles))
	for i := range d.Item.Candles {
		klineData[i] = &kline.Kline{
			Base: event.Base{
				Offset:       int64(i + 1),
				Exchange:     d.Item.Exchange,
				Time:         d.Item.Candles[i].Time,
				Interval:     d.Item.Interval,
				CurrencyPair: d.Item.Pair,
				AssetType:    d.Item.Asset,
			},
			Open:   d.Item.Candles[i].Open,
			High:   d.Item.Candles[i].High,
			Low:    d.Item.Candles[i].Low,
			Close:  d.Item.Candles[i].Close,
			Volume: d.Item.Candles[i].Volume,
		}
		d.addedTimes[d.Item.Candles[i].Time] = true
	}
	d.SetStream(klineData)
	d.SortStream()
	return nil
}

// Append adds a candle item to the data stream and sorts it to ensure it is all in order
func (d *DataFromKline) Append(ki *gctkline.Item) {
	if d.addedTimes == nil {
		d.addedTimes = make(map[time.Time]bool)
	}
	var klineData []common.DataEventHandler
	var gctCandles []gctkline.Candle
	for i := range ki.Candles {
		if _, ok := d.addedTimes[ki.Candles[i].Time]; !ok {
			gctCandles = append(gctCandles, ki.Candles[i])
			d.addedTimes[ki.Candles[i].Time] = true
		}
	}
	var candleTimes []time.Time

	for i := range gctCandles {
		klineData = append(klineData, &kline.Kline{
			Base: event.Base{
				Offset:       int64(i + 1),
				Exchange:     ki.Exchange,
				Time:         gctCandles[i].Time,
				Interval:     ki.Interval,
				CurrencyPair: ki.Pair,
				AssetType:    ki.Asset,
			},
			Open:   gctCandles[i].Open,
			High:   gctCandles[i].High,
			Low:    gctCandles[i].Low,
			Close:  gctCandles[i].Close,
			Volume: gctCandles[i].Volume,
		})
		candleTimes = append(candleTimes, gctCandles[i].Time)
	}
	log.Debugf(log.BackTester, "appending %v candle intervals: %v", len(gctCandles), candleTimes)
	d.AppendStream(klineData...)
	d.SortStream()
}

// StreamOpen returns all Open prices from the beginning until the current iteration
func (d *DataFromKline) StreamOpen() []float64 {
	s := d.GetStream()
	o := d.Offset()

	ret := make([]float64, o)
	for x := range s[:o] {
		ret[x] = s[x].(*kline.Kline).Open
	}
	return ret
}

// StreamHigh returns all High prices from the beginning until the current iteration
func (d *DataFromKline) StreamHigh() []float64 {
	s := d.GetStream()
	o := d.Offset()

	ret := make([]float64, o)
	for x := range s[:o] {
		ret[x] = s[x].(*kline.Kline).High
	}
	return ret
}

// StreamLow returns all Low prices from the beginning until the current iteration
func (d *DataFromKline) StreamLow() []float64 {
	s := d.GetStream()
	o := d.Offset()

	ret := make([]float64, o)
	for x := range s[:o] {
		ret[x] = s[x].(*kline.Kline).Low
	}
	return ret
}

// StreamClose returns all Close prices from the beginning until the current iteration
func (d *DataFromKline) StreamClose() []float64 {
	s := d.GetStream()
	o := d.Offset()

	ret := make([]float64, o)
	for x := range s[:o] {
		ret[x] = s[x].(*kline.Kline).Close
	}
	return ret
}

// StreamVol returns all Volume prices from the beginning until the current iteration
func (d *DataFromKline) StreamVol() []float64 {
	s := d.GetStream()
	o := d.Offset()

	ret := make([]float64, o)
	for x := range s[:o] {
		ret[x] = s[x].(*kline.Kline).Volume
	}
	return ret
}
