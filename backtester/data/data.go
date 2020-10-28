package data

import (
	"sort"

	"github.com/shopspring/decimal"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	portfolio "github.com/thrasher-corp/gocryptotrader/backtester/datahandler"
)

// Load specified data into Candle format
// this is empty and loader types will have their own implementation
func (d *Data) Load() error {
	return nil
}

// Reset loaded data to blank state
func (d *Data) Reset() {
	d.latest = nil
	d.offset = 0
	d.stream = nil
}

// Stream will return entire data list
func (d *Data) Stream() []portfolio.DataEventHandler {
	return d.stream
}

// Next will return the next event in the list and also shift the offset one
func (d *Data) Next() (dh portfolio.DataEventHandler, ok bool) {
	if len(d.stream) <= d.offset {
		return nil, false
	}

	ret := d.stream[d.offset]
	d.offset++
	d.latest = ret
	return ret, true
}

// History will return all previous data events that have happened
func (d *Data) History() []portfolio.DataEventHandler {
	return d.stream[:d.offset]
}

// Latest will return latest data event
func (d *Data) Latest() portfolio.DataEventHandler {
	return d.latest
}

func (d *Data) List() []portfolio.DataEventHandler {
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

func (d *Data) StreamOpen() []float64 {
	return []float64{}
}

func (d *Data) StreamHigh() []float64 {
	return []float64{}
}

func (d *Data) StreamLow() []float64 {
	return []float64{}
}
func (d *Data) StreamClose() []float64 {
	return []float64{}
}

func (d *Data) StreamVol() []float64 {
	return []float64{}
}

func (d *Data) Offset() int {
	return d.offset
}

func (c *Candle) DataType() portfolio.DataType {
	return DataTypeCandle
}

func (c *Candle) LatestPrice() float64 {
	return c.Close
}

func (t *Tick) LatestPrice() float64 {
	bid := decimal.NewFromFloat(t.Bid)
	ask := decimal.NewFromFloat(t.Ask)
	diff := decimal.New(2, 0)
	latest, _ := bid.Add(ask).Div(diff).Round(common.DecimalPlaces).Float64()
	return latest
}

func (t *Tick) DataType() portfolio.DataType {
	return DataTypeTick
}

func (t *Tick) Spread() float64 {
	return t.Bid - t.Ask
}
