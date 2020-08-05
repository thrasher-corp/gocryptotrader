package backtest

import "github.com/thrasher-corp/gocryptotrader/exchanges/kline"

type DataFromKlineItem struct {
	kline kline.Item

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
	return nil
}

func (d *DataFromKlineItem) History() []DataEvent {
	return d.stream[:d.offset]
}

func (d *DataFromKlineItem) Latest() DataEvent {
	return d.latest
}

func (d *DataFromKlineItem) Load() {
	var candles []Candle

	for x := range d.kline.Candles {

	}
}