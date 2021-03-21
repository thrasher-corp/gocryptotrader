package data

import (
	"sort"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// Setup creates a basic map
func (h *HandlerPerCurrency) Setup() {
	if h.data == nil {
		h.data = make(map[string]map[asset.Item]map[currency.Pair]Handler)
	}
}

// SetDataForCurrency assigns a data Handler to the data map by exchange, asset and currency
func (h *HandlerPerCurrency) SetDataForCurrency(e string, a asset.Item, p currency.Pair, k Handler) {
	if h.data == nil {
		h.Setup()
	}
	e = strings.ToLower(e)
	if h.data[e] == nil {
		h.data[e] = make(map[asset.Item]map[currency.Pair]Handler)
	}
	if h.data[e][a] == nil {
		h.data[e][a] = make(map[currency.Pair]Handler)
	}
	h.data[e][a][p] = k
}

// GetAllData returns all set data in the data map
func (h *HandlerPerCurrency) GetAllData() map[string]map[asset.Item]map[currency.Pair]Handler {
	return h.data
}

// GetDataForCurrency returns the Handler for a specific exchange, asset, currency
func (h *HandlerPerCurrency) GetDataForCurrency(e string, a asset.Item, p currency.Pair) Handler {
	return h.data[e][a][p]
}

// Reset returns the struct to defaults
func (h *HandlerPerCurrency) Reset() {
	h.data = nil
}

// Reset loaded data to blank state
func (b *Base) Reset() {
	b.latest = nil
	b.offset = 0
	b.stream = nil
}

// GetStream will return entire data list
func (b *Base) GetStream() []common.DataEventHandler {
	return b.stream
}

// Offset returns the current iteration of candle data the backtester is assessing
func (b *Base) Offset() int {
	return b.offset
}

// SetStream sets the data stream for candle analysis
func (b *Base) SetStream(s []common.DataEventHandler) {
	b.stream = s
}

// AppendStream appends new datas onto the stream, however, will not
// add duplicates. Used for live analysis
func (b *Base) AppendStream(s ...common.DataEventHandler) {
	for i := range s {
		if s[i] == nil {
			continue
		}
		b.stream = append(b.stream, s[i])
	}
}

// Next will return the next event in the list and also shift the offset one
func (b *Base) Next() (dh common.DataEventHandler) {
	if len(b.stream) <= b.offset {
		return nil
	}

	ret := b.stream[b.offset]
	b.offset++
	b.latest = ret
	return ret
}

// History will return all previous data events that have happened
func (b *Base) History() []common.DataEventHandler {
	return b.stream[:b.offset]
}

// Latest will return latest data event
func (b *Base) Latest() common.DataEventHandler {
	return b.latest
}

// List returns all future data events from the current iteration
// ill-advised to use this in strategies because you don't know the future in real life
func (b *Base) List() []common.DataEventHandler {
	return b.stream[b.offset:]
}

// SortStream sorts the stream by timestamp
func (b *Base) SortStream() {
	sort.Slice(b.stream, func(i, j int) bool {
		b1 := b.stream[i]
		b2 := b.stream[j]

		return b1.GetTime().Before(b2.GetTime())
	})
}
