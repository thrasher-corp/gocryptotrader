package kline

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/kline"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// HasDataAtTime verifies checks the underlying range data
// To determine whether there is any candle data present at the time provided
func (d *DataFromKline) HasDataAtTime(t time.Time) bool {
	if d.Base.IsLive() {
		return true
	}
	if d.RangeHolder == nil {
		return false
	}
	return d.RangeHolder.HasDataAtDate(t)
}

// Load sets the candle data to the stream for processing
func (d *DataFromKline) Load() error {
	if d.addedTimes == nil {
		d.addedTimes = make(map[int64]bool)
	}
	if len(d.Item.Candles) == 0 {
		return errNoCandleData
	}

	klineData := make([]data.Event, len(d.Item.Candles))
	for i := range d.Item.Candles {
		newKline := &kline.Kline{
			Base: &event.Base{
				Offset:         int64(i + 1),
				Exchange:       d.Item.Exchange,
				Time:           d.Item.Candles[i].Time.UTC(),
				Interval:       d.Item.Interval,
				CurrencyPair:   d.Item.Pair,
				AssetType:      d.Item.Asset,
				UnderlyingPair: d.Item.UnderlyingPair,
			},
			Open:             decimal.NewFromFloat(d.Item.Candles[i].Open),
			High:             decimal.NewFromFloat(d.Item.Candles[i].High),
			Low:              decimal.NewFromFloat(d.Item.Candles[i].Low),
			Close:            decimal.NewFromFloat(d.Item.Candles[i].Close),
			Volume:           decimal.NewFromFloat(d.Item.Candles[i].Volume),
			ValidationIssues: d.Item.Candles[i].ValidationIssues,
		}
		klineData[i] = newKline
		d.addedTimes[d.Item.Candles[i].Time.UTC().UnixNano()] = true
	}

	d.SetStream(klineData)
	d.SortStream()
	return nil
}

// AppendResults adds a candle item to the data stream and sorts it to ensure it is all in order
func (d *DataFromKline) AppendResults(ki *gctkline.Item) {
	if ki == nil {
		return
	}
	if !d.Item.EqualSource(ki) {
		return
	}
	if d.addedTimes == nil {
		d.addedTimes = make(map[int64]bool)
	}
	for i := range d.Item.Candles {
		d.addedTimes[d.Item.Candles[i].Time.UTC().UnixNano()] = true
	}
	var gctCandles []gctkline.Candle
	for i := range ki.Candles {
		if _, ok := d.addedTimes[ki.Candles[i].Time.UTC().UnixNano()]; !ok {
			gctCandles = append(gctCandles, ki.Candles[i])
			d.addedTimes[ki.Candles[i].Time.UTC().UnixNano()] = true
		}
	}
	if len(gctCandles) == 0 {
		return
	}
	for i := range gctCandles {
		d.Item.Candles = append(d.Item.Candles, gctCandles[i])
	}
	d.Item.RemoveDuplicates()
	if d.RangeHolder != nil {
		// offline data check when there is a known range
		// live data does not need this
		d.RangeHolder.SetHasDataFromCandles(d.Item.Candles)
	}
	err := d.Load()
	if err != nil {
		log.Errorf(common.Data, "unable to load candles %v", err)
	}
	d.SortStream()
}

// StreamOpen returns all Open prices from the beginning until the current iteration
func (d *DataFromKline) StreamOpen() []decimal.Decimal {
	s := d.GetStream()
	o := d.Offset()

	ret := make([]decimal.Decimal, o)
	for x := range s[:o] {
		if val, ok := s[x].(*kline.Kline); ok {
			ret[x] = val.Open
		} else {
			log.Errorf(common.Data, "incorrect data loaded into stream")
		}
	}
	return ret
}

// StreamHigh returns all High prices from the beginning until the current iteration
func (d *DataFromKline) StreamHigh() []decimal.Decimal {
	s := d.GetStream()
	o := d.Offset()

	ret := make([]decimal.Decimal, o)
	for x := range s[:o] {
		if val, ok := s[x].(*kline.Kline); ok {
			ret[x] = val.High
		} else {
			log.Errorf(common.Data, "incorrect data loaded into stream")
		}
	}
	return ret
}

// StreamLow returns all Low prices from the beginning until the current iteration
func (d *DataFromKline) StreamLow() []decimal.Decimal {
	s := d.GetStream()
	o := d.Offset()

	ret := make([]decimal.Decimal, o)
	for x := range s[:o] {
		if val, ok := s[x].(*kline.Kline); ok {
			ret[x] = val.Low
		} else {
			log.Errorf(common.Data, "incorrect data loaded into stream")
		}
	}
	return ret
}

// StreamClose returns all Close prices from the beginning until the current iteration
func (d *DataFromKline) StreamClose() []decimal.Decimal {
	s := d.GetStream()
	o := d.Offset()

	ret := make([]decimal.Decimal, o)
	for x := range s[:o] {
		if val, ok := s[x].(*kline.Kline); ok {
			ret[x] = val.Close
		} else {
			log.Errorf(common.Data, "incorrect data loaded into stream")
		}
	}
	return ret
}

// StreamVol returns all Volume prices from the beginning until the current iteration
func (d *DataFromKline) StreamVol() []decimal.Decimal {
	s := d.GetStream()
	o := d.Offset()

	ret := make([]decimal.Decimal, o)
	for x := range s[:o] {
		if val, ok := s[x].(*kline.Kline); ok {
			ret[x] = val.Volume
		} else {
			log.Errorf(common.Data, "incorrect data loaded into stream")
		}
	}
	return ret
}
