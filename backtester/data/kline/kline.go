package kline

import (
	"errors"

	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	portfolio "github.com/thrasher-corp/gocryptotrader/backtester/datahandler"
	"github.com/thrasher-corp/gocryptotrader/backtester/event"
)

func (c *Candle) DataType() portfolio.DataType {
	return data.DataTypeCandle
}

func (c *Candle) LatestPrice() float64 {
	return c.Close
}

func (d *DataFromKline) Load() error {
	if len(d.Item.Candles) == 0 {
		return errors.New("no candle data provided")
	}

	klineData := make([]portfolio.DataEventHandler, len(d.Item.Candles))
	for i := range d.Item.Candles {
		klineData[i] = &Candle{
			Event: event.Event{
				Time: d.Item.Candles[i].Time, CurrencyPair: d.Item.Pair,
			},
			Open:   d.Item.Candles[i].Open,
			High:   d.Item.Candles[i].High,
			Low:    d.Item.Candles[i].Low,
			Close:  d.Item.Candles[i].Close,
			Volume: d.Item.Candles[i].Volume,
		}
	}
	d.SetStream(klineData)
	d.SortStream()
	return nil
}

func (d *DataFromKline) StreamOpen() []float64 {
	s := d.GetStream()
	o := d.GetOffset()

	ret := make([]float64, o)
	for x := range s[:o] {
		ret[x] = s[x].(*Candle).Open
	}
	return ret
}

func (d *DataFromKline) StreamHigh() []float64 {
	s := d.GetStream()
	o := d.GetOffset()

	ret := make([]float64, o)
	for x := range s[:o] {
		ret[x] = s[x].(*Candle).High
	}
	return ret
}

func (d *DataFromKline) StreamLow() []float64 {
	s := d.GetStream()
	o := d.GetOffset()

	ret := make([]float64, o)
	for x := range s[:o] {
		ret[x] = s[x].(*Candle).Low
	}
	return ret
}

func (d *DataFromKline) StreamClose() []float64 {
	s := d.GetStream()
	o := d.GetOffset()

	ret := make([]float64, o)
	for x := range s[:o] {
		ret[x] = s[x].(*Candle).Close
	}
	return ret
}

func (d *DataFromKline) StreamVol() []float64 {
	s := d.GetStream()
	o := d.GetOffset()

	ret := make([]float64, o)
	for x := range s[:o] {
		ret[x] = s[x].(*Candle).Volume
	}
	return ret
}
