package kline

import (
	"errors"

	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

type DataFromKline struct {
	Item gctkline.Item
	data.Data
}

func (d *DataFromKline) Load() error {
	if len(d.Item.Candles) == 0 {
		return errors.New("no candle data provided")
	}

	klineData := make([]interfaces.DataEventHandler, len(d.Item.Candles))
	for i := range d.Item.Candles {
		klineData[i] = &kline.Kline{
			Event: event.Event{
				Exchange: d.Item.Exchange,
				Time:     d.Item.Candles[i].Time, CurrencyPair: d.Item.Pair,
				AssetType: d.Item.Asset,
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
		ret[x] = s[x].(*kline.Kline).Open
	}
	return ret
}

func (d *DataFromKline) StreamHigh() []float64 {
	s := d.GetStream()
	o := d.GetOffset()

	ret := make([]float64, o)
	for x := range s[:o] {
		ret[x] = s[x].(*kline.Kline).High
	}
	return ret
}

func (d *DataFromKline) StreamLow() []float64 {
	s := d.GetStream()
	o := d.GetOffset()

	ret := make([]float64, o)
	for x := range s[:o] {
		ret[x] = s[x].(*kline.Kline).Low
	}
	return ret
}

func (d *DataFromKline) StreamClose() []float64 {
	s := d.GetStream()
	o := d.GetOffset()

	ret := make([]float64, o)
	for x := range s[:o] {
		ret[x] = s[x].(*kline.Kline).Close
	}
	return ret
}

func (d *DataFromKline) StreamVol() []float64 {
	s := d.GetStream()
	o := d.GetOffset()

	ret := make([]float64, o)
	for x := range s[:o] {
		ret[x] = s[x].(*kline.Kline).Volume
	}
	return ret
}
