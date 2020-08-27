package backtest

import (
	"errors"

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

	data := make([]DataEventHandler, len(d.Item.Candles))
	for i := range d.Item.Candles {
		data[i] = &Candle{
			Event: Event{
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

func (d *DataFromKline) StreamClose() []float64 {
	ret := make([]float64, len(d.stream))
	for x := range d.stream[d.offset:] {
		ret[x] = d.stream[x].(*Candle).Close
	}
	return ret
}