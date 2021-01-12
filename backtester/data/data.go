package data

import (
	"sort"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

func (d *HandlerPerCurrency) Setup() {
	if d.data == nil {
		d.data = make(map[string]map[asset.Item]map[currency.Pair]Handler)
	}
}

func (d *HandlerPerCurrency) SetDataForCurrency(e string, a asset.Item, p currency.Pair, k Handler) {
	if d.data == nil {
		d.Setup()
	}
	e = strings.ToLower(e)
	if d.data[e] == nil {
		d.data[e] = make(map[asset.Item]map[currency.Pair]Handler)
	}
	if d.data[e][a] == nil {
		d.data[e][a] = make(map[currency.Pair]Handler)
	}
	d.data[e][a][p] = k
}

func (d *HandlerPerCurrency) GetAllData() map[string]map[asset.Item]map[currency.Pair]Handler {
	return d.data
}

func (d *HandlerPerCurrency) GetDataForCurrency(e string, a asset.Item, p currency.Pair) Handler {
	return d.data[e][a][p]
}

func (d *HandlerPerCurrency) Reset() {
	d.data = nil
}

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
func (d *Data) GetStream() []common.DataEventHandler {
	return d.stream
}

func (d *Data) Offset() int {
	return d.offset
}

func (d *Data) SetStream(s []common.DataEventHandler) {
	d.stream = s
}

// AppendStream appends new datas onto the stream, however, will not
// add duplicates. Used for live analysis
func (d *Data) AppendStream(s ...common.DataEventHandler) {
	for i := range s {
		if s[i] == nil {
			continue
		}
		d.stream = append(d.stream, s[i])
	}
}

// Next will return the next event in the list and also shift the offset one
func (d *Data) Next() (dh common.DataEventHandler, ok bool) {
	if len(d.stream) <= d.offset {
		return nil, false
	}

	ret := d.stream[d.offset]
	d.offset++
	d.latest = ret
	return ret, true
}

// History will return all previous data events that have happened
func (d *Data) History() []common.DataEventHandler {
	return d.stream[:d.offset]
}

// Latest will return latest data event
func (d *Data) Latest() common.DataEventHandler {
	return d.latest
}

func (d *Data) List() []common.DataEventHandler {
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
