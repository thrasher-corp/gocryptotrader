package data

import (
	"errors"
	"sort"

	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

func (d Data) Load(in []kline.Candle) error {
	if len(in) < 1 {
		return errors.New("no candle data provided")
	}
	return nil
}

func (d Data) Next() (out Event, ok bool) {
	if len(d.stream) == 0 {
		return out, false
	}

	out, d.stream = d.stream[0], d.stream[1:]
	d.history = append(d.history, out)

	d.latest[out.Pair().String()] = out
	d.list[out.Pair().String()] = append(d.list[out.Pair().String()], out)

	return out, true
}

func (d Data) Stream() []Event {
	return d.stream
}

func (d Data) History() []Event {
	return d.history
}

func (d Data) Latest(symbol string) Event {
	return d.latest[symbol]
}

func (d Data) List(symbol string) []Event {
	return d.list[symbol]
}

func (d Data) Reset() {
	d.latest = nil
	d.list = nil
	d.stream = d.history
	d.history = nil
}

func (d Data) Sort() {
	sort.Slice(d.stream, func(i, j int) bool {
		return d.stream[i].Time().Before(d.stream[i].Time())
	})
}

func (t *Tick) Price() float64 {
	return (t.Bid + t.Ask) / float64(2)
}

func (t *Tick) Spread() float64 {
	return t.Bid - t.Ask
}
