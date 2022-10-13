package data

import (
	"errors"
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
	m1, ok := h.data[e]
	if !ok {
		m1 = make(map[asset.Item]map[*currency.Item]map[*currency.Item]Handler)
		h.data[e] = m1
	}

	m2, ok := m1[a]
	if !ok {
		m2 = make(map[*currency.Item]map[*currency.Item]Handler)
		m1[a] = m2
	}

	m3, ok := m2[p.Base.Item]
	if !ok {
		m3 = make(map[*currency.Item]Handler)
		m2[p.Base.Item] = m3
	}

	m3[p.Quote.Item] = k
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
		return nil, fmt.Errorf("%s %s %s %w", exch, a, p, ErrHandlerNotFound)
	}
	return handler, nil
}

// Reset returns the struct to defaults
func (h *HandlerPerCurrency) Reset() {
	hi := &HandlerPerCurrency{data: make(map[string]map[asset.Item]map[*currency.Item]map[*currency.Item]Handler)}
	*h = *hi
}

// Reset loaded data to blank state
func (b *Base) Reset() {
	*b = Base{}
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
	b.stream = make([]Event, len(s))
	// due to the Next() function, we cannot take
	// stream offsets as is, and we re-set them
	for i := range s {
		b.stream[i] = s[i]
		b.stream[i].SetOffset(int64(i + 1))
	}
}

var errNothingToAdd = errors.New("passed in empty void")
var errInvalidEventSupplied = errors.New("invalid event supplied")

// AppendStream appends new datas onto the stream, however, will not
// add duplicates. Used for live analysis
func (b *Base) AppendStream(s ...Event) error {
	if len(s) == 0 {
		return errNothingToAdd
	}
	updatedStream := make([]Event, 0, len(s))
candles:
	for x := range s {
		if !b.equalSource(s[x]) {
			return errInvalidEventSupplied
		}
		for y := range b.stream {
			if s[x].GetTime().Equal(b.stream[y].GetTime()) {
				continue candles
			}
		}
		updatedStream = append(updatedStream, s[x])
	}
	if len(updatedStream) == 0 {
		return nil
	}
	b.stream = append(b.stream, updatedStream...)
	b.SortStream()
	for i := range b.stream {
		b.stream[i].SetOffset(int64(i + 1))
	}
	return nil
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
