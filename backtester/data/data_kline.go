package data

import (
	"errors"

	portfolio "github.com/thrasher-corp/gocryptotrader/backtester/datahandler"
	"github.com/thrasher-corp/gocryptotrader/backtester/event"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

type DataFromKline struct {
	Item kline.Item

	Data
}

func (d *DataFromKline) Load() error {
	if len(d.Item.Candles) == 0 {
		return errors.New("no candle data provided")
	}

	data := make([]portfolio.DataEventHandler, len(d.Item.Candles))
	for i := range d.Item.Candles {
		data[i] = &Candle{
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
	d.stream = data
	d.SortStream()
	return nil
}

func (d *DataFromKline) StreamOpen() []float64 {
	ret := make([]float64, len(d.stream))
	for x := range d.stream {
		ret[x] = d.stream[x].(*Candle).Open
	}
	return ret
}

func (d *DataFromKline) StreamHigh() []float64 {
	ret := make([]float64, len(d.stream))
	for x := range d.stream {
		ret[x] = d.stream[x].(*Candle).High
	}
	return ret
}

func (d *DataFromKline) StreamLow() []float64 {
	ret := make([]float64, len(d.stream))
	for x := range d.stream {
		ret[x] = d.stream[x].(*Candle).Low
	}
	return ret
}

func (d *DataFromKline) StreamClose() []float64 {
	ret := make([]float64, len(d.stream))
	for x := range d.stream {
		ret[x] = d.stream[x].(*Candle).Close
	}
	return ret
}

func (d *DataFromKline) StreamVol() []float64 {
	ret := make([]float64, len(d.stream))
	for x := range d.stream {
		ret[x] = d.stream[x].(*Candle).Volume
	}
	return ret
}
