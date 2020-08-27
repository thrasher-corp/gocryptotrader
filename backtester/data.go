package backtest

import (
	"sort"

	"github.com/shopspring/decimal"
)

func (d *Data) Load() error {
	return nil
}

func (d *Data) Reset() {
	d.latest = nil
	d.offset = 0
	d.stream = nil
}

func (d *Data) Stream() []DataEventHandler {
	return d.stream
}

func (d *Data) Next() (dh DataEventHandler, ok bool) {
	if len(d.stream) <= d.offset {
		return nil, false
	}

	ret := d.stream[d.offset]
	d.offset++
	d.latest = ret
	return ret, true
}

func (d *Data) History() []DataEventHandler {
	return d.stream[:d.offset]
}

func (d *Data) Latest() DataEventHandler {
	return d.latest
}

func (d *Data) List() []DataEventHandler {
	return d.stream[d.offset:]
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

func (c *Candle) DataType() DataType {
	return DataTypeCandle
}

func (c *Candle) LatestPrice() float64 {
	return c.Close
}

func (t *Tick) LatestPrice() float64 {
	bid := decimal.NewFromFloat(t.Bid)
	ask := decimal.NewFromFloat(t.Ask)
	diff := decimal.New(2, 0)
	latest, _ := bid.Add(ask).Div(diff).Round(DP).Float64()
	return latest
}

func (t *Tick) DataType() DataType {
	return DataTypeTick
}

func (t *Tick) Spread() float64 {
	return t.Bid - t.Ask
}
