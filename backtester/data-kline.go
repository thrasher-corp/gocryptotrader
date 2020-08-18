package backtest

import (
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

func (d *DataFromKlineItem) Load() {
	var candles []*Candle
	for x := range d.Item.Candles {
		candles = append(candles, &Candle{
			Open:   d.Item.Candles[x].Open,
			High:   d.Item.Candles[x].High,
			Low:    d.Item.Candles[x].Low,
			Close:  d.Item.Candles[x].Close,
			Volume: d.Item.Candles[x].Volume,
		})
	}

	list := make([]DataEvent, len(candles))
	for i := range candles {
		list[i] = candles[i]
	}
	d.SetStream(list)
}

func (d *DataFromKlineItem) SortStream() {
	sort.Slice(d.stream, func(i, j int) bool {
		b1 := d.stream[i]
		b2 := d.stream[j]

		return b1.Time().Before(b2.Time())
	})
}

func (d *DataFromKlineItem) updateLatest(event DataEvent) {
	if d.latest == nil {
		d.latest = make(map[string]Data)
	}

	d.latest = event
}

func (d *Data) updateList(event DataEvent) {
	if d.list == nil {
		d.list = make(map[string][]DataEvent)
	}

	d.list[event.Pair()] = append(d.list[event.Pair()], event)
}
