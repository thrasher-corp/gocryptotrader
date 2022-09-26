package data

import (
	"fmt"
	"sort"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// Setup creates a basic map
func (h *HandlerPerCurrency) Setup() {
	if h.data == nil {
		h.data = make(map[string]map[asset.Item]map[*currency.Item]map[*currency.Item]Handler)
	}
}

// SetDataForCurrency assigns a data Handler to the data map by exchange, asset and currency
func (h *HandlerPerCurrency) SetDataForCurrency(e string, a asset.Item, p currency.Pair, k Handler) {
	if h.data == nil {
		h.Setup()
	}
	e = strings.ToLower(e)
	if h.data[e] == nil {
		h.data[e] = make(map[asset.Item]map[*currency.Item]map[*currency.Item]Handler)
	}
	if h.data[e][a] == nil {
		h.data[e][a] = make(map[*currency.Item]map[*currency.Item]Handler)
	}
	if h.data[e][a][p.Base.Item] == nil {
		h.data[e][a][p.Base.Item] = make(map[*currency.Item]Handler)
	}
	h.data[e][a][p.Base.Item][p.Quote.Item] = k
}

// GetAllData returns all set data in the data map
func (h *HandlerPerCurrency) GetAllData() map[string]map[asset.Item]map[*currency.Item]map[*currency.Item]Handler {
	return h.data
}

// GetDataForCurrency returns the Handler for a specific exchange, asset, currency
func (h *HandlerPerCurrency) GetDataForCurrency(ev common.Event) (Handler, error) {
	if ev == nil {
		return nil, common.ErrNilEvent
	}
	exch := ev.GetExchange()
	a := ev.GetAssetType()
	p := ev.Pair()
	handler, ok := h.data[exch][a][p.Base.Item][p.Quote.Item]
	if !ok {
		return nil, fmt.Errorf("%s %s %s %w", ev.GetExchange(), ev.GetAssetType(), ev.Pair(), ErrHandlerNotFound)
	}
	return handler, nil
}

// Reset returns the struct to defaults
func (h *HandlerPerCurrency) Reset() {
	if h == nil {
		return
	}
	h.data = nil
}

// Reset loaded data to blank state
func (b *Base) Reset() {
	if b == nil {
		return
	}
	b.latest = nil
	b.offset = 0
	b.stream = nil
}

// GetStream will return entire data list
func (b *Base) GetStream() []Event {
	return b.stream
}

// Offset returns the current iteration of candle data the backtester is assessing
func (b *Base) Offset() int64 {
	return b.offset
}

// SetStream sets the data stream for candle analysis
func (b *Base) SetStream(s []Event) {
	b.stream = s
	// due to the Next() function, we cannot take
	// stream offsets as is, and we re-set them
	for i := range b.stream {
		b.stream[i].SetOffset(int64(i + 1))
	}
}

// AppendStream appends new datas onto the stream, however, will not
// add duplicates. Used for live analysis
func (b *Base) AppendStream(s ...Event) {
candles:
	for x := range s {
		if s[x] == nil || !b.equalSource(s[x]) {
			continue
		}
		for y := range b.stream {
			if s[x].GetTime().Equal(b.stream[y].GetTime()) {
				continue candles
			}
		}
		b.stream = append(b.stream, s[x])
	}
	for i := range b.stream {
		b.stream[i].SetOffset(int64(i + 1))
	}
	b.SortStream()
}

// equalSource verifies that incoming data matches
// internal source
func (b *Base) equalSource(s Event) bool {
	if b == nil || s == nil {
		return false
	}
	if s.GetExchange() == "" || !s.GetAssetType().IsValid() || s.Pair().IsEmpty() {
		return false
	}
	if len(b.stream) == 0 {
		return true
	}
	return s.GetExchange() == b.stream[0].GetExchange() &&
		s.GetAssetType() == b.stream[0].GetAssetType() &&
		s.Pair().Equal(b.stream[0].Pair())
}

// Next will return the next event in the list and also shift the offset one
func (b *Base) Next() Event {
	if int64(len(b.stream)) <= b.offset {
		return nil
	}
	ret := b.stream[b.offset]
	b.offset++
	b.latest = ret
	return ret
}

// History will return all previous data events that have happened
func (b *Base) History() []Event {
	return b.stream[:b.offset]
}

// Latest will return latest data event
func (b *Base) Latest() Event {
	if b.latest == nil && int64(len(b.stream)) >= b.offset+1 {
		b.latest = b.stream[b.offset]
	}
	return b.latest
}

// List returns all future data events from the current iteration
// ill-advised to use this in strategies because you don't know the future in real life
func (b *Base) List() []Event {
	return b.stream[b.offset:]
}

// IsLastEvent determines whether the latest event is the last event
// for live data, this will be false, as all appended data is the latest available data
// and this signal cannot be completely relied upon
func (b *Base) IsLastEvent() bool {
	return b.latest != nil && b.latest.GetOffset() == int64(len(b.stream)) && !b.isLiveData
}

// SortStream sorts the stream by timestamp
func (b *Base) SortStream() {
	sort.Slice(b.stream, func(i, j int) bool {
		return b.stream[i].GetTime().Before(b.stream[j].GetTime())
	})
}

// IsLive returns if the data source is a live one
// less scrutiny on checks is required on live data sourcing
func (b *Base) IsLive() bool {
	return b.isLiveData
}

// SetLive sets if the data source is a live one
// less scrutiny on checks is required on live data sourcing
func (b *Base) SetLive(isLive bool) {
	b.isLiveData = isLive
}
