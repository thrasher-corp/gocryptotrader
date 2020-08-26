package backtest

import (
	"sort"

	"github.com/thrasher-corp/gocryptotrader/currency"
)

func (d *Data) Load() error {
	return nil
}

func (d *Data) Reset() {
	d.latest = nil
	d.list = nil
	d.stream = d.streamHistory
	d.streamHistory = nil
}

func (d *Data) Stream() []DataEventHandler {
	return d.stream
}

func (d *Data) Next() (dh DataEventHandler, ok bool) {
	if len(d.stream) == 0 {
		return dh, false
	}

	dh = d.stream[0]
	d.stream = d.stream[1:]
	d.streamHistory = append(d.streamHistory, dh)

	d.updateLatest(dh)
	d.updateList(dh)

	return dh, true
}

func (d *Data) History() []DataEventHandler {
	return d.streamHistory
}

func (d *Data) Latest(pair currency.Pair) DataEventHandler {
	return d.latest[pair.String()]
}

func (d *Data) List(pair currency.Pair) []DataEventHandler {
	return d.list[pair.String()]
}

func (d *Data) SortStream() {
	sort.Slice(d.stream, func(i, j int) bool {
		b1 := d.stream[i]
		b2 := d.stream[j]

		if b1.GetTime().Equal(b2.GetTime()) {
			return b1.Pair().String() < b2.Pair().String()
		}
		return b1.GetTime().Before(b2.GetTime())
	})
}

func (d *Data) updateLatest(event DataEventHandler) {
	if d.latest == nil {
		d.latest = make(map[string]DataEventHandler)
	}

	d.latest[event.Pair().String()] = event
}

func (d *Data) updateList(event DataEventHandler) {
	if d.list == nil {
		d.list = make(map[string][]DataEventHandler)
	}

	d.list[event.Pair().String()] = append(d.list[event.Pair().String()], event)
}
