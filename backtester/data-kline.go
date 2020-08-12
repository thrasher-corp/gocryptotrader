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

func (d *DataFromKlineItem) Load() {
	var candles []*Candle
	for x := range d.kline.Candles {
		candles = append(candles, &Candle{
			Open:   d.kline.Candles[x].Open,
			High:   d.kline.Candles[x].High,
			Low:    d.kline.Candles[x].Low,
			Close:  d.kline.Candles[x].Close,
			Volume: d.kline.Candles[x].Volume,
		})
	}

	list := make([]DataEvent, len(candles))
	for i := range candles {
		list[i] = candles[i]
	}
	d.SetStream(list)
}