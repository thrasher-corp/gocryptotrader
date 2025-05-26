package data

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

const (
	exch = "binance"
	a    = asset.Spot
)

var p = currency.NewBTCUSD()

type fakeEvent struct {
	secretID int64
	*event.Base
}

type fakeHandler struct{}

func TestSetDataForCurrency(t *testing.T) {
	t.Parallel()
	d := HandlerHolder{}
	err := d.SetDataForCurrency(exch, a, p, nil)
	assert.NoError(t, err)

	if d.data == nil {
		t.Error("expected not nil")
	}
	if d.data[key.ExchangePairAsset{
		Exchange: exch,
		Base:     p.Base.Item,
		Quote:    p.Quote.Item,
		Asset:    a,
	}] != nil {
		t.Error("expected nil")
	}
}

func TestGetAllData(t *testing.T) {
	t.Parallel()
	d := HandlerHolder{}
	err := d.SetDataForCurrency(exch, a, p, nil)
	assert.NoError(t, err)

	err = d.SetDataForCurrency(exch, a, currency.NewPair(currency.BTC, currency.DOGE), nil)
	assert.NoError(t, err)

	result, err := d.GetAllData()
	assert.NoError(t, err)

	if len(result) != 2 {
		t.Error("expected 2")
	}
}

func TestGetDataForCurrency(t *testing.T) {
	t.Parallel()
	d := HandlerHolder{}
	err := d.SetDataForCurrency(exch, a, p, &fakeHandler{})
	assert.NoError(t, err)

	err = d.SetDataForCurrency(exch, a, currency.NewPair(currency.BTC, currency.DOGE), nil)
	assert.NoError(t, err)

	_, err = d.GetDataForCurrency(nil)
	if !errors.Is(err, common.ErrNilEvent) {
		t.Errorf("received '%v' expected '%v'", err, common.ErrNilEvent)
	}

	_, err = d.GetDataForCurrency(&fakeEvent{Base: &event.Base{
		Exchange:     "lol",
		AssetType:    asset.USDTMarginedFutures,
		CurrencyPair: currency.NewPair(currency.EMB, currency.DOGE),
	}})
	if !errors.Is(err, ErrHandlerNotFound) {
		t.Errorf("received '%v' expected '%v'", err, ErrHandlerNotFound)
	}

	_, err = d.GetDataForCurrency(&fakeEvent{Base: &event.Base{
		Exchange:     exch,
		AssetType:    a,
		CurrencyPair: p,
	}})
	assert.NoError(t, err)
}

func TestReset(t *testing.T) {
	t.Parallel()
	d := &HandlerHolder{}
	err := d.SetDataForCurrency(exch, a, p, nil)
	assert.NoError(t, err)

	err = d.SetDataForCurrency(exch, a, currency.NewPair(currency.BTC, currency.DOGE), nil)
	assert.NoError(t, err)

	err = d.Reset()
	assert.NoError(t, err)

	if d.data == nil {
		t.Error("expected a map")
	}
	d = nil
	err = d.Reset()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
}

func TestBaseReset(t *testing.T) {
	t.Parallel()
	b := &Base{offset: 1}
	err := b.Reset()
	assert.NoError(t, err)

	if b.offset != 0 {
		t.Errorf("received '%v' expected '%v'", b.offset, 0)
	}
	b = nil
	err = b.Reset()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
}

func TestGetStream(t *testing.T) {
	t.Parallel()
	b := &Base{}
	resp, err := b.GetStream()
	assert.NoError(t, err)

	if len(resp) != 0 {
		t.Errorf("received '%v' expected '%v'", len(resp), 0)
	}
	b.stream = []Event{
		&fakeEvent{
			Base: &event.Base{
				Offset: 2048,
				Time:   time.Now(),
			},
		},
		&fakeEvent{
			Base: &event.Base{
				Offset: 1337,
				Time:   time.Now().Add(-time.Hour),
			},
		},
	}
	resp, err = b.GetStream()
	assert.NoError(t, err)

	if len(resp) != 2 {
		t.Errorf("received '%v' expected '%v'", len(resp), 2)
	}

	b = nil
	_, err = b.GetStream()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
}

func TestOffset(t *testing.T) {
	t.Parallel()
	b := &Base{}
	o, err := b.Offset()
	assert.NoError(t, err)

	if o != 0 {
		t.Errorf("received '%v' expected '%v'", o, 0)
	}
	b.offset = 1337
	o, err = b.Offset()
	assert.NoError(t, err)

	if o != 1337 {
		t.Errorf("received '%v' expected '%v'", o, 1337)
	}

	b = nil
	_, err = b.Offset()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
}

func TestSetStream(t *testing.T) {
	t.Parallel()
	b := &Base{}
	err := b.SetStream(nil)
	assert.NoError(t, err)

	if len(b.stream) != 0 {
		t.Errorf("received '%v' expected '%v'", len(b.stream), 0)
	}
	cp := currency.NewBTCUSD()
	err = b.SetStream([]Event{
		&fakeEvent{
			Base: &event.Base{
				Offset:       2048,
				Time:         time.Now(),
				Exchange:     "test",
				AssetType:    asset.Spot,
				CurrencyPair: cp,
			},
		},
		&fakeEvent{
			Base: &event.Base{
				Offset:       1337,
				Time:         time.Now().Add(-time.Hour),
				Exchange:     "test",
				AssetType:    asset.Spot,
				CurrencyPair: cp,
			},
		},
	})
	assert.NoError(t, err)

	if len(b.stream) != 2 {
		t.Fatalf("received '%v' expected '%v'", len(b.stream), 2)
	}
	if b.stream[0].GetOffset() != 1 {
		t.Errorf("received '%v' expected '%v'", b.stream[0].GetOffset(), 1)
	}

	misMatchEvent := &fakeEvent{
		Base: &event.Base{
			Exchange:     "mismatch",
			CurrencyPair: currency.NewPair(currency.BTC, currency.DOGE),
			AssetType:    asset.Futures,
		},
	}
	err = b.SetStream([]Event{misMatchEvent})
	if !errors.Is(err, ErrInvalidEventSupplied) {
		t.Fatalf("received '%v' expected '%v'", err, ErrInvalidEventSupplied)
	}

	misMatchEvent.Time = time.Now()
	err = b.SetStream([]Event{misMatchEvent})
	if !errors.Is(err, errMisMatchedEvent) {
		t.Fatalf("received '%v' expected '%v'", err, errMisMatchedEvent)
	}

	err = b.SetStream([]Event{nil})
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Fatalf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}

	b = nil
	err = b.SetStream(nil)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
}

func TestNext(t *testing.T) {
	t.Parallel()
	b := &Base{}
	cp := currency.NewBTCUSD()
	err := b.SetStream([]Event{
		&fakeEvent{
			Base: &event.Base{
				Offset:       2048,
				Time:         time.Now(),
				Exchange:     "test",
				AssetType:    asset.Spot,
				CurrencyPair: cp,
			},
		},
		&fakeEvent{
			Base: &event.Base{
				Offset:       1337,
				Time:         time.Now().Add(-time.Hour),
				Exchange:     "test",
				AssetType:    asset.Spot,
				CurrencyPair: cp,
			},
		},
	})
	assert.NoError(t, err)

	resp, err := b.Next()
	assert.NoError(t, err)

	if resp != b.stream[0] {
		t.Errorf("received '%v' expected '%v'", resp, b.stream[0])
	}
	if b.offset != 1 {
		t.Errorf("received '%v' expected '%v'", b.offset, 1)
	}
	_, err = b.Next()
	assert.NoError(t, err)

	resp, err = b.Next()
	if !errors.Is(err, ErrEndOfData) {
		t.Errorf("received '%v' expected '%v'", err, ErrEndOfData)
	}
	if resp != nil {
		t.Errorf("received '%v' expected '%v'", resp, nil)
	}

	b = nil
	_, err = b.Next()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
}

func TestHistory(t *testing.T) {
	t.Parallel()
	b := &Base{}
	cp := currency.NewBTCUSD()
	err := b.SetStream([]Event{
		&fakeEvent{
			Base: &event.Base{
				Offset:       2048,
				Time:         time.Now(),
				Exchange:     "test",
				AssetType:    asset.Spot,
				CurrencyPair: cp,
			},
		},
		&fakeEvent{
			Base: &event.Base{
				Offset:       1337,
				Time:         time.Now().Add(-time.Hour),
				Exchange:     "test",
				AssetType:    asset.Spot,
				CurrencyPair: cp,
			},
		},
	})
	assert.NoError(t, err)

	resp, err := b.History()
	assert.NoError(t, err)

	if len(resp) != 0 {
		t.Errorf("received '%v' expected '%v'", len(resp), 0)
	}

	_, err = b.Next()
	assert.NoError(t, err)

	resp, err = b.History()
	assert.NoError(t, err)

	if len(resp) != 1 {
		t.Errorf("received '%v' expected '%v'", len(resp), 1)
	}

	b = nil
	_, err = b.History()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
}

func TestLatest(t *testing.T) {
	t.Parallel()
	b := &Base{}
	cp := currency.NewBTCUSD()
	err := b.SetStream([]Event{
		&fakeEvent{
			Base: &event.Base{
				Offset:       2048,
				Time:         time.Now(),
				Exchange:     "test",
				AssetType:    asset.Spot,
				CurrencyPair: cp,
			},
		},
		&fakeEvent{
			Base: &event.Base{
				Offset:       1337,
				Time:         time.Now().Add(-time.Hour),
				Exchange:     "test",
				AssetType:    asset.Spot,
				CurrencyPair: cp,
			},
		},
	})
	assert.NoError(t, err)

	resp, err := b.Latest()
	assert.NoError(t, err)

	if resp != b.stream[0] {
		t.Errorf("received '%v' expected '%v'", resp, b.stream[0])
	}
	_, err = b.Next()
	assert.NoError(t, err)

	resp, err = b.Latest()
	assert.NoError(t, err)

	if resp != b.stream[0] {
		t.Errorf("received '%v' expected '%v'", resp, b.stream[0])
	}

	_, err = b.Next()
	assert.NoError(t, err)

	resp, err = b.Latest()
	assert.NoError(t, err)

	if resp != b.stream[1] {
		t.Errorf("received '%v' expected '%v'", resp, b.stream[1])
	}

	b = nil
	_, err = b.Latest()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
}

func TestList(t *testing.T) {
	t.Parallel()
	b := &Base{}
	cp := currency.NewBTCUSD()
	err := b.SetStream([]Event{
		&fakeEvent{
			Base: &event.Base{
				Offset:       2048,
				Time:         time.Now(),
				Exchange:     "test",
				AssetType:    asset.Spot,
				CurrencyPair: cp,
			},
		},
		&fakeEvent{
			Base: &event.Base{
				Offset:       1337,
				Time:         time.Now().Add(-time.Hour),
				Exchange:     "test",
				AssetType:    asset.Spot,
				CurrencyPair: cp,
			},
		},
	})
	assert.NoError(t, err)

	list, err := b.List()
	assert.NoError(t, err)

	if len(list) != 2 {
		t.Errorf("received '%v' expected '%v'", len(list), 2)
	}

	b = nil
	_, err = b.List()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
}

func TestIsLastEvent(t *testing.T) {
	t.Parallel()
	b := &Base{}
	cp := currency.NewBTCUSD()
	err := b.SetStream([]Event{
		&fakeEvent{
			Base: &event.Base{
				Offset:       2048,
				Time:         time.Now(),
				Exchange:     "test",
				AssetType:    asset.Spot,
				CurrencyPair: cp,
			},
		},
		&fakeEvent{
			Base: &event.Base{
				Offset:       1337,
				Time:         time.Now().Add(-time.Hour),
				Exchange:     "test",
				AssetType:    asset.Spot,
				CurrencyPair: cp,
			},
		},
	})
	assert.NoError(t, err)

	b.latest = b.stream[0]
	b.offset = b.stream[0].GetOffset()
	isLastEvent, err := b.IsLastEvent()
	assert.NoError(t, err)

	if isLastEvent {
		t.Errorf("received '%v' expected '%v'", false, true)
	}

	b.isLiveData = true
	isLastEvent, err = b.IsLastEvent()
	assert.NoError(t, err)

	if isLastEvent {
		t.Errorf("received '%v' expected '%v'", false, true)
	}

	b = nil
	_, err = b.IsLastEvent()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
}

func TestIsLive(t *testing.T) {
	t.Parallel()
	b := &Base{}
	isLive, err := b.IsLive()
	assert.NoError(t, err)

	if isLive {
		t.Error("expected false")
	}
	b.isLiveData = true
	isLive, err = b.IsLive()
	assert.NoError(t, err)

	if !isLive {
		t.Error("expected true")
	}

	b = nil
	_, err = b.IsLive()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
}

func TestSetLive(t *testing.T) {
	t.Parallel()
	b := &Base{}
	err := b.SetLive(true)
	assert.NoError(t, err)

	if !b.isLiveData {
		t.Error("expected true")
	}

	err = b.SetLive(false)
	assert.NoError(t, err)

	if b.isLiveData {
		t.Error("expected false")
	}

	b = nil
	err = b.SetLive(false)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
}

func TestAppendStream(t *testing.T) {
	t.Parallel()
	b := &Base{}
	e := &fakeEvent{
		Base: &event.Base{},
	}
	err := b.AppendStream(e)
	if !errors.Is(err, ErrInvalidEventSupplied) {
		t.Errorf("received '%v' expected '%v'", err, ErrInvalidEventSupplied)
	}
	if len(b.stream) != 0 {
		t.Errorf("received '%v' expected '%v'", len(b.stream), 0)
	}
	tt := time.Now().Add(-time.Hour)
	cp := currency.NewBTCUSD()
	e.Exchange = "b"
	e.AssetType = asset.Spot
	e.CurrencyPair = cp
	err = b.AppendStream(e)
	if !errors.Is(err, ErrInvalidEventSupplied) {
		t.Fatalf("received '%v' expected '%v'", err, ErrInvalidEventSupplied)
	}

	e.Time = tt
	err = b.AppendStream(e, e)
	require.NoError(t, err)

	if len(b.stream) != 1 {
		t.Errorf("received '%v' expected '%v'", len(b.stream), 1)
	}

	err = b.AppendStream(e)
	require.NoError(t, err)

	if len(b.stream) != 1 {
		t.Errorf("received '%v' expected '%v'", len(b.stream), 1)
	}

	err = b.AppendStream(&fakeEvent{
		Base: &event.Base{
			Exchange:     "b",
			AssetType:    asset.Spot,
			CurrencyPair: cp,
			Time:         time.Now(),
		},
	})
	require.NoError(t, err)

	if len(b.stream) != 2 {
		t.Errorf("received '%v' expected '%v'", len(b.stream), 2)
	}

	misMatchEvent := &fakeEvent{
		Base: &event.Base{
			Exchange:     "mismatch",
			CurrencyPair: currency.NewPair(currency.BTC, currency.DOGE),
			AssetType:    asset.Futures,
			Time:         tt,
		},
	}
	err = b.AppendStream(misMatchEvent)
	if !errors.Is(err, errMisMatchedEvent) {
		t.Fatalf("received '%v' expected '%v'", err, errMisMatchedEvent)
	}
	if len(b.stream) != 2 {
		t.Errorf("received '%v' expected '%v'", len(b.stream), 2)
	}

	err = b.AppendStream(nil)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Fatalf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
	if len(b.stream) != 2 {
		t.Errorf("received '%v' expected '%v'", len(b.stream), 2)
	}

	err = b.AppendStream()
	if !errors.Is(err, errNothingToAdd) {
		t.Fatalf("received '%v' expected '%v'", err, errNothingToAdd)
	}
	if len(b.stream) != 2 {
		t.Errorf("received '%v' expected '%v'", len(b.stream), 2)
	}

	b = nil
	err = b.AppendStream()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
}

func TestFirst(t *testing.T) {
	t.Parallel()
	var id1 int64 = 1
	var id2 int64 = 2
	var id3 int64 = 3
	e := Events{
		fakeEvent{secretID: id1},
		fakeEvent{secretID: id2},
		fakeEvent{secretID: id3},
	}

	first, err := e.First()
	assert.NoError(t, err)

	if first.GetOffset() != id1 {
		t.Errorf("received '%v' expected '%v'", first.GetOffset(), id1)
	}
}

func TestLast(t *testing.T) {
	t.Parallel()
	var id1 int64 = 1
	var id2 int64 = 2
	var id3 int64 = 3
	e := Events{
		fakeEvent{secretID: id1},
		fakeEvent{secretID: id2},
		fakeEvent{secretID: id3},
	}

	last, err := e.Last()
	assert.NoError(t, err)

	if last.GetOffset() != id3 {
		t.Errorf("received '%v' expected '%v'", last.GetOffset(), id1)
	}
}

// methods that satisfy the common.Event interface
func (f fakeEvent) GetOffset() int64 {
	if f.secretID > 0 {
		return f.secretID
	}
	return f.Offset
}

func (f fakeEvent) SetOffset(o int64) {
	f.Offset = o
}

func (f fakeEvent) IsEvent() bool {
	return false
}

func (f fakeEvent) GetTime() time.Time {
	return f.Base.Time
}

func (f fakeEvent) Pair() currency.Pair {
	return currency.NewBTCUSD()
}

func (f fakeEvent) GetExchange() string {
	return f.Exchange
}

func (f fakeEvent) GetInterval() gctkline.Interval {
	return gctkline.Interval(time.Minute)
}

func (f fakeEvent) GetAssetType() asset.Item {
	return f.AssetType
}

func (f fakeEvent) GetReason() string {
	return strings.Join(f.Reasons, ",")
}

func (f fakeEvent) AppendReason(string) {
}

func (f fakeEvent) GetClosePrice() decimal.Decimal {
	return decimal.Zero
}

func (f fakeEvent) GetHighPrice() decimal.Decimal {
	return decimal.Zero
}

func (f fakeEvent) GetLowPrice() decimal.Decimal {
	return decimal.Zero
}

func (f fakeEvent) GetOpenPrice() decimal.Decimal {
	return decimal.Zero
}

func (f fakeEvent) GetVolume() decimal.Decimal {
	return decimal.Zero
}

func (f fakeEvent) GetUnderlyingPair() currency.Pair {
	return f.Pair()
}

func (f fakeEvent) AppendReasonf(string, ...any) {}

func (f fakeEvent) GetBase() *event.Base {
	return &event.Base{}
}

func (f fakeEvent) GetConcatReasons() string {
	return ""
}

func (f fakeEvent) GetReasons() []string {
	return nil
}

func (f fakeHandler) Load() error {
	return nil
}

func (f fakeHandler) AppendStream(...Event) error {
	return nil
}

func (f fakeHandler) GetBase() Base {
	return Base{}
}

func (f fakeHandler) Next() (Event, error) {
	return nil, nil
}

func (f fakeHandler) GetStream() (Events, error) {
	return nil, nil
}

func (f fakeHandler) History() (Events, error) {
	return nil, nil
}

func (f fakeHandler) Latest() (Event, error) {
	return nil, nil
}

func (f fakeHandler) List() (Events, error) {
	return nil, nil
}

func (f fakeHandler) IsLastEvent() (bool, error) {
	return false, nil
}

func (f fakeHandler) Offset() (int64, error) {
	return 0, nil
}

func (f fakeHandler) StreamOpen() ([]decimal.Decimal, error) {
	return nil, nil
}

func (f fakeHandler) StreamHigh() ([]decimal.Decimal, error) {
	return nil, nil
}

func (f fakeHandler) StreamLow() ([]decimal.Decimal, error) {
	return nil, nil
}

func (f fakeHandler) StreamClose() ([]decimal.Decimal, error) {
	return nil, nil
}

func (f fakeHandler) StreamVol() ([]decimal.Decimal, error) {
	return nil, nil
}

func (f fakeHandler) HasDataAtTime(time.Time) (bool, error) {
	return false, nil
}

func (f fakeHandler) Reset() error {
	return nil
}

func (f fakeHandler) GetDetails() (string, asset.Item, currency.Pair, error) {
	return "", asset.Empty, currency.EMPTYPAIR, nil
}
