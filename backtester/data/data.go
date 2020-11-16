package data

import (
	"sort"

	"github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
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
func (d *Data) GetStream() []interfaces.DataEventHandler {
	return d.stream
}

func (d *Data) GetOffset() int {
	return d.offset
}

func (d *Data) SetStream(s []interfaces.DataEventHandler) {
	d.stream = s
}

// AppendStream appends new datas onto the stream, however, will not
// add duplicates. Used for live analysis
func (d *Data) AppendStream(s ...interfaces.DataEventHandler) {
	for i := range s {
		for j := range d.stream {
			if d.stream[j].GetTime().Before(s[i].GetTime()) ||
				d.stream[j].GetTime().Equal(s[i].GetTime()) {
				continue
			}
			d.stream = append(d.stream, s[i])
		}
	}
}

// Next will return the next event in the list and also shift the offset one
func (d *Data) Next() (dh interfaces.DataEventHandler, ok bool) {
	if len(d.stream) <= d.offset {
		return nil, false
	}

	ret := d.stream[d.offset]
	d.offset++
	d.latest = ret
	return ret, true
}

// History will return all previous data events that have happened
func (d *Data) History() []interfaces.DataEventHandler {
	return d.stream[:d.offset]
}

// Latest will return latest data event
func (d *Data) Latest() interfaces.DataEventHandler {
	return d.latest
}

func (d *Data) List() []interfaces.DataEventHandler {
	return d.stream[d.offset:]
}

func (d *Data) SortStream() {
	sort.Slice(d.stream, func(i, j int) bool {
		b1 := d.stream[i]
		b2 := d.stream[j]

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
