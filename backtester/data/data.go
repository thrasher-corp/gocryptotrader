package data

import (
	"fmt"
	"sort"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// NewHandlerHolder returns a new HandlerHolder
func NewHandlerHolder() *HandlerHolder {
	return &HandlerHolder{
		data: make(map[key.ExchangeAssetPair]Handler),
	}
}

// SetDataForCurrency assigns a Data Handler to the Data map by exchange, asset and currency
func (h *HandlerHolder) SetDataForCurrency(e string, a asset.Item, p currency.Pair, k Handler) error {
	if h == nil {
		return fmt.Errorf("%w handler holder", gctcommon.ErrNilPointer)
	}
	h.m.Lock()
	defer h.m.Unlock()
	if h.data == nil {
		h.data = make(map[key.ExchangeAssetPair]Handler)
	}
	e = strings.ToLower(e)
	h.data[key.NewExchangeAssetPair(e, a, p)] = k
	return nil
}

// GetAllData returns all set Data in the Data map
func (h *HandlerHolder) GetAllData() ([]Handler, error) {
	if h == nil {
		return nil, fmt.Errorf("%w handler holder", gctcommon.ErrNilPointer)
	}
	h.m.Lock()
	defer h.m.Unlock()
	resp := make([]Handler, 0, len(h.data))
	for _, handler := range h.data {
		resp = append(resp, handler)
	}
	return resp, nil
}

// GetDataForCurrency returns the Handler for a specific exchange, asset, currency
func (h *HandlerHolder) GetDataForCurrency(ev common.Event) (Handler, error) {
	if h == nil {
		return nil, fmt.Errorf("%w handler holder", gctcommon.ErrNilPointer)
	}
	if ev == nil {
		return nil, common.ErrNilEvent
	}
	h.m.Lock()
	defer h.m.Unlock()
	exch := ev.GetExchange()
	a := ev.GetAssetType()
	p := ev.Pair()
	handler, ok := h.data[key.NewExchangeAssetPair(exch, a, p)]
	if !ok {
		return nil, fmt.Errorf("%s %s %s %w", exch, a, p, ErrHandlerNotFound)
	}
	return handler, nil
}

// Reset returns the struct to defaults
func (h *HandlerHolder) Reset() error {
	if h == nil {
		return gctcommon.ErrNilPointer
	}
	h.m.Lock()
	defer h.m.Unlock()
	h.data = make(map[key.ExchangeAssetPair]Handler)
	return nil
}

// GetDetails returns data about the Base Holder
func (b *Base) GetDetails() (string, asset.Item, currency.Pair, error) {
	if b == nil {
		return "", asset.Empty, currency.EMPTYPAIR, fmt.Errorf("%w base", gctcommon.ErrNilPointer)
	}
	b.m.Lock()
	defer b.m.Unlock()
	return b.latest.GetExchange(), b.latest.GetAssetType(), b.latest.Pair(), nil
}

// Reset loaded Data to blank state
func (b *Base) Reset() error {
	if b == nil {
		return gctcommon.ErrNilPointer
	}
	b.m.Lock()
	defer b.m.Unlock()
	b.stream = nil
	b.latest = nil
	b.offset = 0
	b.isLiveData = false
	return nil
}

// GetStream will return entire Data list
func (b *Base) GetStream() (Events, error) {
	if b == nil {
		return nil, fmt.Errorf("%w Base", gctcommon.ErrNilPointer)
	}
	b.m.Lock()
	defer b.m.Unlock()
	stream := make([]Event, len(b.stream))
	copy(stream, b.stream)

	return stream, nil
}

// Offset returns the current iteration of candle Data the backtester is assessing
func (b *Base) Offset() (int64, error) {
	if b == nil {
		return 0, fmt.Errorf("%w Base", gctcommon.ErrNilPointer)
	}
	b.m.Lock()
	defer b.m.Unlock()
	return b.offset, nil
}

// SetStream sets the Data stream for candle analysis
func (b *Base) SetStream(s []Event) error {
	if b == nil {
		return fmt.Errorf("%w Base", gctcommon.ErrNilPointer)
	}
	b.m.Lock()
	defer b.m.Unlock()

	sort.Slice(s, func(i, j int) bool {
		return s[i].GetTime().Before(s[j].GetTime())
	})
	for x := range s {
		if s[x] == nil {
			return fmt.Errorf("%w Event", gctcommon.ErrNilPointer)
		}
		if s[x].GetExchange() == "" || !s[x].GetAssetType().IsValid() || s[x].Pair().IsEmpty() || s[x].GetTime().IsZero() {
			return ErrInvalidEventSupplied
		}
		if len(b.stream) > 0 {
			if s[x].GetExchange() != b.stream[0].GetExchange() ||
				s[x].GetAssetType() != b.stream[0].GetAssetType() ||
				!s[x].Pair().Equal(b.stream[0].Pair()) {
				return fmt.Errorf("%w cannot set base stream from %v %v %v to %v %v %v", errMismatchedEvent, s[x].GetExchange(), s[x].GetAssetType(), s[x].Pair(), b.stream[0].GetExchange(), b.stream[0].GetAssetType(), b.stream[0].Pair())
			}
		}
		// due to the Next() function, we cannot take
		// stream offsets as is, and we re-set them
		s[x].SetOffset(int64(x) + 1)
	}

	b.stream = s
	return nil
}

// AppendStream appends new datas onto the stream, however, will not
// add duplicates. Used for live analysis
func (b *Base) AppendStream(s ...Event) error {
	if b == nil {
		return fmt.Errorf("%w Base", gctcommon.ErrNilPointer)
	}
	if len(s) == 0 {
		return errNothingToAdd
	}
	b.m.Lock()
	defer b.m.Unlock()
candles:
	for x := range s {
		if s[x] == nil {
			return fmt.Errorf("%w Event", gctcommon.ErrNilPointer)
		}
		if s[x].GetExchange() == "" || !s[x].GetAssetType().IsValid() || s[x].Pair().IsEmpty() || s[x].GetTime().IsZero() {
			return ErrInvalidEventSupplied
		}
		if len(b.stream) > 0 {
			if s[x].GetExchange() != b.stream[0].GetExchange() ||
				s[x].GetAssetType() != b.stream[0].GetAssetType() ||
				!s[x].Pair().Equal(b.stream[0].Pair()) {
				return fmt.Errorf("%w %v %v %v received  %v %v %v", errMismatchedEvent, b.stream[0].GetExchange(), b.stream[0].GetAssetType(), b.stream[0].Pair(), s[x].GetExchange(), s[x].GetAssetType(), s[x].Pair())
			}
			// todo change b.stream to map
			for y := len(b.stream) - 1; y >= 0; y-- {
				if s[x].GetTime().Equal(b.stream[y].GetTime()) {
					continue candles
				}
			}
		}

		b.stream = append(b.stream, s[x])
	}

	sort.Slice(b.stream, func(i, j int) bool {
		return b.stream[i].GetTime().Before(b.stream[j].GetTime())
	})
	for i := range b.stream {
		b.stream[i].SetOffset(int64(i) + 1)
	}
	return nil
}

// Next will return the next event in the list and also shift the offset one
func (b *Base) Next() (Event, error) {
	if b == nil {
		return nil, fmt.Errorf("%w Base", gctcommon.ErrNilPointer)
	}
	b.m.Lock()
	defer b.m.Unlock()
	if int64(len(b.stream)) <= b.offset {
		return nil, fmt.Errorf("%w data length %v offset %v", ErrEndOfData, len(b.stream), b.offset)
	}
	ret := b.stream[b.offset]
	b.offset++
	b.latest = ret
	return ret, nil
}

// History will return all previous Data events that have happened
func (b *Base) History() (Events, error) {
	if b == nil {
		return nil, fmt.Errorf("%w Base", gctcommon.ErrNilPointer)
	}
	b.m.Lock()
	defer b.m.Unlock()

	stream := make([]Event, len(b.stream[:b.offset]))
	copy(stream, b.stream[:b.offset])

	return stream, nil
}

// Latest will return latest Data event
func (b *Base) Latest() (Event, error) {
	if b == nil {
		return nil, fmt.Errorf("%w Base", gctcommon.ErrNilPointer)
	}
	b.m.Lock()
	defer b.m.Unlock()

	if b.latest == nil && int64(len(b.stream)) >= b.offset+1 {
		b.latest = b.stream[b.offset]
	}
	return b.latest, nil
}

// List returns all future Data events from the current iteration
// ill-advised to use this in strategies because you don't know the future in real life
func (b *Base) List() (Events, error) {
	if b == nil {
		return nil, fmt.Errorf("%w Base", gctcommon.ErrNilPointer)
	}
	b.m.Lock()
	defer b.m.Unlock()

	stream := make([]Event, len(b.stream[b.offset:]))
	copy(stream, b.stream[b.offset:])

	return stream, nil
}

// IsLastEvent determines whether the latest event is the last event
// for live Data, this will be false, as all appended Data is the latest available Data
// and this signal cannot be completely relied upon
func (b *Base) IsLastEvent() (bool, error) {
	if b == nil {
		return false, fmt.Errorf("%w Base", gctcommon.ErrNilPointer)
	}
	b.m.Lock()
	defer b.m.Unlock()

	return b.latest != nil && b.latest.GetOffset() == int64(len(b.stream)) && !b.isLiveData,
		nil
}

// IsLive returns if the Data source is a live one
// less scrutiny on checks is required on live Data sourcing
func (b *Base) IsLive() (bool, error) {
	if b == nil {
		return false, fmt.Errorf("%w Base", gctcommon.ErrNilPointer)
	}
	b.m.Lock()
	defer b.m.Unlock()

	return b.isLiveData, nil
}

// SetLive sets if the Data source is a live one
// less scrutiny on checks is required on live Data sourcing
func (b *Base) SetLive(isLive bool) error {
	if b == nil {
		return fmt.Errorf("%w Base", gctcommon.ErrNilPointer)
	}
	b.m.Lock()
	defer b.m.Unlock()

	b.isLiveData = isLive
	return nil
}

// First returns the first element of a slice
func (e Events) First() (Event, error) {
	if len(e) == 0 {
		return nil, ErrEmptySlice
	}
	return e[0], nil
}

// Last returns the last element of a slice
func (e Events) Last() (Event, error) {
	if len(e) == 0 {
		return nil, ErrEmptySlice
	}
	return e[len(e)-1], nil
}
