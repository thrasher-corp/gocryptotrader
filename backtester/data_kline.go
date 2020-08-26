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

	var data []DataEventHandler
	for i := range d.Item.Candles {
		data = append(data, &Candle{
			Event: Event{
				Time: d.Item.Candles[i].Time, CurrencyPair: d.Item.Pair,
			},
			Open:   d.Item.Candles[i].Open,
			High:   d.Item.Candles[i].High,
			Low:    d.Item.Candles[i].Low,
			Close:  d.Item.Candles[i].Close,
			Volume: d.Item.Candles[i].Volume,
		})
	}
	d.stream = data
	d.SortStream()
	return nil
}
