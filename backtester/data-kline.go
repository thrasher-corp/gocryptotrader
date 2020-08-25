package backtest

import (
	"errors"
	"sort"

	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

type DataFromKlineItem struct {
	Item kline.Item

	latest DataEvent
	stream []DataEvent
	offset int
}

func (d *DataFromKlineItem) Reset() {
	d.latest = nil
	d.offset = 0
}

func (d *DataFromKlineItem) Next() (DataEvent, bool) {
	if len(d.stream) <= d.offset {
		return nil, false
	}

	ret := d.stream[d.offset]
	d.offset++
	d.latest = ret

	return ret, true
}

func (d *DataFromKlineItem) Stream() []DataEvent {
	return d.stream[d.offset:]
}

func (d *DataFromKlineItem) History() []DataEvent {
	return d.stream[:d.offset]
}

func (d *DataFromKlineItem) Latest() DataEvent {
	return d.latest
}

func (d *DataFromKlineItem) SetStream(stream []DataEvent) {
	d.stream = stream
}

func (d *DataFromKlineItem) Load() error {
	if len(d.Item.Candles) == 0 {
		return errors.New("no candle data provided")
	}

	list := make([]DataEvent, len(d.Item.Candles))
	for i := range d.Item.Candles {
		list[i] = &Candle{
			timestamp: d.Item.Candles[i].Time,
			Open:      d.Item.Candles[i].Open,
			High:      d.Item.Candles[i].High,
			Low:       d.Item.Candles[i].Low,
			Close:     d.Item.Candles[i].Close,
			Volume:    d.Item.Candles[i].Volume,
		}
	}
	d.stream = list
	return nil
}

func (d *DataFromKlineItem) SortStream() {
	sort.Slice(d.stream, func(i, j int) bool {
		b1 := d.stream[i]
		b2 := d.stream[j]

		return b1.Time().Before(b2.Time())
	})
}

func (d *DataFromKlineItem) updateLatest(event DataEvent) {
	d.latest = event
}
