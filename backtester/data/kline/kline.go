package kline

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/kline"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

// NewDataFromKline returns a new struct
func NewDataFromKline() *DataFromKline {
	return &DataFromKline{
		Base: &data.Base{},
	}
}

// HasDataAtTime verifies checks the underlying range data
// To determine whether there is any candle data present at the time provided
func (d *DataFromKline) HasDataAtTime(t time.Time) (bool, error) {
	isLive, err := d.Base.IsLive()
	if err != nil {
		return false, err
	}
	if isLive {
		var s []data.Event
		s, err = d.GetStream()
		if err != nil {
			return false, err
		}
		for i := range s {
			if s[i].GetTime().Equal(t) {
				return true, nil
			}
		}
		return false, nil
	}
	if d.RangeHolder == nil {
		return false, fmt.Errorf("%w RangeHolder", gctcommon.ErrNilPointer)
	}
	return d.RangeHolder.HasDataAtDate(t), nil
}

// Load sets the candle data to the stream for processing
func (d *DataFromKline) Load() error {
	if d.Item == nil || len(d.Item.Candles) == 0 {
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
	}

	return d.SetStream(klineData)
}

// AppendResults adds a candle item to the data stream and sorts it to ensure it is all in order
func (d *DataFromKline) AppendResults(ki *gctkline.Item) error {
	if ki == nil {
		return fmt.Errorf("%w kline item", gctcommon.ErrNilPointer)
	}
	err := d.Item.EqualSource(ki)
	if err != nil {
		return err
	}
	var gctCandles []gctkline.Candle
	stream, err := d.Base.GetStream()
	if err != nil {
		return err
	}
candleLoop:
	for x := range ki.Candles {
		for y := range stream {
			if stream[y].GetTime().Equal(ki.Candles[x].Time) {
				continue candleLoop
			}
		}
		gctCandles = append(gctCandles, ki.Candles[x])
	}
	if len(gctCandles) == 0 {
		return nil
	}
	klineData := make([]data.Event, len(gctCandles))
	for i := range gctCandles {
		d.Item.Candles = append(d.Item.Candles, gctCandles[i])
		newKline := &kline.Kline{
			Base: &event.Base{
				Exchange:       d.Item.Exchange,
				Interval:       d.Item.Interval,
				CurrencyPair:   d.Item.Pair,
				AssetType:      d.Item.Asset,
				UnderlyingPair: d.Item.UnderlyingPair,
				Time:           gctCandles[i].Time.UTC(),
			},
			Open:   decimal.NewFromFloat(gctCandles[i].Open),
			High:   decimal.NewFromFloat(gctCandles[i].High),
			Low:    decimal.NewFromFloat(gctCandles[i].Low),
			Close:  decimal.NewFromFloat(gctCandles[i].Close),
			Volume: decimal.NewFromFloat(gctCandles[i].Volume),
		}
		klineData[i] = newKline
	}
	err = d.AppendStream(klineData...)
	if err != nil {
		return err
	}

	d.Item.RemoveDuplicates()
	d.Item.SortCandlesByTimestamp(false)
	if d.RangeHolder != nil {
		d.RangeHolder, err = gctkline.CalculateCandleDateRanges(d.Item.Candles[0].Time, d.Item.Candles[len(d.Item.Candles)-1].Time.Add(d.Item.Interval.Duration()), d.Item.Interval, d.RangeHolder.Limit)
		if err != nil {
			return err
		}
		// offline data check when there is a known range
		// live data does not need this
		return d.RangeHolder.SetHasDataFromCandles(d.Item.Candles)
	}
	return nil
}

// StreamOpen returns all Open prices from the beginning until the current iteration
func (d *DataFromKline) StreamOpen() ([]decimal.Decimal, error) {
	s, err := d.History()
	if err != nil {
		return nil, err
	}

	ret := make([]decimal.Decimal, len(s))
	for x := range s {
		ret[x] = s[x].GetOpenPrice()
	}
	return ret, nil
}

// StreamHigh returns all High prices from the beginning until the current iteration
func (d *DataFromKline) StreamHigh() ([]decimal.Decimal, error) {
	s, err := d.History()
	if err != nil {
		return nil, err
	}

	ret := make([]decimal.Decimal, len(s))
	for x := range s {
		ret[x] = s[x].GetHighPrice()
	}
	return ret, nil
}

// StreamLow returns all Low prices from the beginning until the current iteration
func (d *DataFromKline) StreamLow() ([]decimal.Decimal, error) {
	s, err := d.History()
	if err != nil {
		return nil, err
	}

	ret := make([]decimal.Decimal, len(s))
	for x := range s {
		ret[x] = s[x].GetLowPrice()
	}
	return ret, nil
}

// StreamClose returns all Close prices from the beginning until the current iteration
func (d *DataFromKline) StreamClose() ([]decimal.Decimal, error) {
	s, err := d.History()
	if err != nil {
		return nil, err
	}

	ret := make([]decimal.Decimal, len(s))
	for x := range s {
		ret[x] = s[x].GetClosePrice()
	}
	return ret, nil
}

// StreamVol returns all Volume prices from the beginning until the current iteration
func (d *DataFromKline) StreamVol() ([]decimal.Decimal, error) {
	s, err := d.History()
	if err != nil {
		return nil, err
	}

	ret := make([]decimal.Decimal, len(s))
	for x := range s {
		ret[x] = s[x].GetVolume()
	}
	return ret, nil
}
