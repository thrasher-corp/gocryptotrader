package data

import (
	"errors"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

const exch = "binance"
const a = asset.Spot

var p = currency.NewPair(currency.BTC, currency.USD)

type fakeEvent struct {
	*event.Base
}

type fakeHandler struct{}

func TestSetup(t *testing.T) {
	t.Parallel()
	d := HandlerPerCurrency{}
	d.Setup()
	if d.data == nil {
		t.Error("expected not nil")
	}
}

func TestSetDataForCurrency(t *testing.T) {
	t.Parallel()
	d := HandlerPerCurrency{}
	d.SetDataForCurrency(exch, a, p, nil)
	if d.data == nil {
		t.Error("expected not nil")
	}
	if d.data[exch][a][p.Base.Item][p.Quote.Item] != nil {
		t.Error("expected nil")
	}
}

func TestGetAllData(t *testing.T) {
	t.Parallel()
	d := HandlerPerCurrency{}
	d.SetDataForCurrency(exch, a, p, nil)
	d.SetDataForCurrency(exch, a, currency.NewPair(currency.BTC, currency.DOGE), nil)
	result := d.GetAllData()
	if len(result) != 1 {
		t.Error("expected 1")
	}
	if len(result[exch][a][currency.BTC.Item]) != 2 {
		t.Error("expected 2")
	}
}

func TestGetDataForCurrency(t *testing.T) {
	t.Parallel()
	d := HandlerPerCurrency{}
	d.SetDataForCurrency(exch, a, p, &fakeHandler{})

	d.SetDataForCurrency(exch, a, currency.NewPair(currency.BTC, currency.DOGE), nil)

	_, err := d.GetDataForCurrency(nil)
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
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
}

func TestReset(t *testing.T) {
	t.Parallel()
	d := &HandlerPerCurrency{}
	d.SetDataForCurrency(exch, a, p, nil)
	d.SetDataForCurrency(exch, a, currency.NewPair(currency.BTC, currency.DOGE), nil)
	err := d.Reset()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
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
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
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
	b := Base{}
	resp := b.GetStream()
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
	resp = b.GetStream()
	if len(resp) != 2 {
		t.Errorf("received '%v' expected '%v'", len(resp), 2)
	}
}

func TestOffset(t *testing.T) {
	t.Parallel()
	b := Base{}
	o := b.Offset()
	if o != 0 {
		t.Errorf("received '%v' expected '%v'", o, 0)
	}
	b.offset = 1337
	o = b.Offset()
	if o != 1337 {
		t.Errorf("received '%v' expected '%v'", o, 1337)
	}
}

func TestSetStream(t *testing.T) {
	t.Parallel()
	b := Base{}
	b.SetStream(nil)
	if len(b.stream) != 0 {
		t.Errorf("received '%v' expected '%v'", len(b.stream), 0)
	}
	b.SetStream([]Event{
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
	})
	if len(b.stream) != 2 {
		t.Fatalf("received '%v' expected '%v'", len(b.stream), 2)
	}
	if b.stream[0].GetOffset() != 1 {
		t.Errorf("received '%v' expected '%v'", b.stream[0].GetOffset(), 1)
	}
}

func TestNext(t *testing.T) {
	b := Base{}
	b.SetStream([]Event{
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
	})
	resp := b.Next()
	if resp != b.stream[0] {
		t.Errorf("received '%v' expected '%v'", resp, b.stream[0])
	}
	if b.offset != 1 {
		t.Errorf("received '%v' expected '%v'", b.offset, 1)
	}
	_ = b.Next()
	resp = b.Next()
	if resp != nil {
		t.Errorf("received '%v' expected '%v'", resp, nil)
	}
}

func TestHistory(t *testing.T) {
	b := Base{}
	b.SetStream([]Event{
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
	})
	resp := b.History()
	if len(resp) != 0 {
		t.Errorf("received '%v' expected '%v'", len(resp), 0)
	}

	_ = b.Next()
	resp = b.History()
	if len(resp) != 1 {
		t.Errorf("received '%v' expected '%v'", len(resp), 1)
	}
}

func TestLatest(t *testing.T) {
	b := Base{}
	b.SetStream([]Event{
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
	})
	resp := b.Latest()
	if resp != b.stream[0] {
		t.Errorf("received '%v' expected '%v'", resp, b.stream[0])
	}
	_ = b.Next()
	resp = b.Latest()
	if resp != b.stream[0] {
		t.Errorf("received '%v' expected '%v'", resp, b.stream[0])
	}

	_ = b.Next()
	resp = b.Latest()
	if resp != b.stream[1] {
		t.Errorf("received '%v' expected '%v'", resp, b.stream[1])
	}
}

func TestList(t *testing.T) {
	b := Base{}
	b.SetStream([]Event{
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
	})
	list := b.List()
	if len(list) != 2 {
		t.Errorf("received '%v' expected '%v'", len(list), 2)
	}
}

func TestIsLastEvent(t *testing.T) {
	t.Parallel()
	b := Base{}
	b.SetStream([]Event{
		&fakeEvent{
			Base: &event.Base{
				Offset: 1,
				Time:   time.Now(),
			},
		},
	})
	b.latest = b.stream[0]
	b.offset = b.stream[0].GetOffset()
	if !b.IsLastEvent() {
		t.Errorf("received '%v' expected '%v'", false, true)
	}

	b.isLiveData = true
	if b.IsLastEvent() {
		t.Errorf("received '%v' expected '%v'", true, false)
	}
}

func TestIsLive(t *testing.T) {
	t.Parallel()
	b := Base{}
	if b.IsLive() {
		t.Error("expected false")
	}
	b.isLiveData = true
	if !b.IsLive() {
		t.Error("expected true")
	}
}

func TestSetLive(t *testing.T) {
	t.Parallel()
	b := Base{}
	b.SetLive(true)
	if !b.isLiveData {
		t.Error("expected true")
	}

	b.SetLive(false)
	if b.isLiveData {
		t.Error("expected false")
	}
}

func TestAppendResults(t *testing.T) {
	t.Parallel()
	b := Base{}
	validEvent := &fakeEvent{
		Base: &event.Base{},
	}
	b.AppendStream(validEvent)
	if len(b.stream) != 0 {
		t.Errorf("received '%v' expected '%v'", len(b.stream), 0)
	}
	tt := time.Now()
	validEvent.Exchange = "b"
	validEvent.AssetType = asset.Spot
	validEvent.CurrencyPair = currency.NewPair(currency.BTC, currency.USD)
	validEvent.Time = tt
	b.AppendStream(validEvent)
	if len(b.stream) != 1 {
		t.Errorf("received '%v' expected '%v'", len(b.stream), 1)
	}

	b.AppendStream(validEvent)
	if len(b.stream) != 1 {
		t.Errorf("received '%v' expected '%v'", len(b.stream), 1)
	}

	misMatchEvent := &fakeEvent{
		Base: &event.Base{
			Exchange:     "mismatch",
			CurrencyPair: currency.NewPair(currency.BTC, currency.DOGE),
			AssetType:    asset.Futures,
			Time:         tt,
		},
	}
	b.AppendStream(misMatchEvent)
	if len(b.stream) != 1 {
		t.Errorf("received '%v' expected '%v'", len(b.stream), 1)
	}

	b.AppendStream(nil)
	if len(b.stream) != 1 {
		t.Errorf("received '%v' expected '%v'", len(b.stream), 1)
	}

	b.AppendStream()
	if len(b.stream) != 1 {
		t.Errorf("received '%v' expected '%v'", len(b.stream), 1)
	}
}

func TestEqualSource(t *testing.T) {
	t.Parallel()
	b := &Base{}

	err := b.equalSource(nil)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}

	emptyEvent := &fakeEvent{
		Base: &event.Base{},
	}
	err = b.equalSource(emptyEvent)
	if !errors.Is(err, errNothingToAdd) {
		t.Errorf("received '%v' expected '%v'", err, errNothingToAdd)
	}

	validEvent := &fakeEvent{
		Base: &event.Base{
			Exchange:     "b",
			CurrencyPair: currency.NewPair(currency.BTC, currency.USD),
			AssetType:    asset.Spot,
		},
	}
	err = b.equalSource(validEvent)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	b.stream = append(b.stream, validEvent)
	err = b.equalSource(validEvent)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	misMatchEvent := &fakeEvent{
		Base: &event.Base{
			Exchange:     "mismatch",
			CurrencyPair: currency.NewPair(currency.BTC, currency.DOGE),
			AssetType:    asset.Futures,
		},
	}
	err = b.equalSource(misMatchEvent)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
}

// methods that satisfy the common.Event interface
func (f fakeEvent) GetOffset() int64 {
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
	return currency.NewPair(currency.BTC, currency.USD)
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

func (f fakeEvent) AppendReasonf(s string, i ...interface{}) {}

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

func (f fakeHandler) AppendStream(s ...Event) error {
	return nil
}

func (f fakeHandler) GetBase() Base {
	return Base{}
}

func (f fakeHandler) Next() Event {
	return nil
}

func (f fakeHandler) GetStream() []Event {
	return nil
}

func (f fakeHandler) History() []Event {
	return nil
}

func (f fakeHandler) Latest() Event {
	return nil
}

func (f fakeHandler) List() []Event {
	return nil
}

func (f fakeHandler) IsLastEvent() bool {
	return false
}

func (f fakeHandler) Offset() int64 {
	return 0
}

func (f fakeHandler) StreamOpen() []decimal.Decimal {
	return nil
}

func (f fakeHandler) StreamHigh() []decimal.Decimal {
	return nil
}

func (f fakeHandler) StreamLow() []decimal.Decimal {
	return nil
}

func (f fakeHandler) StreamClose() []decimal.Decimal {
	return nil
}

func (f fakeHandler) StreamVol() []decimal.Decimal {
	return nil
}

func (f fakeHandler) HasDataAtTime(t time.Time) bool {
	return false
}

func (f fakeHandler) Reset() error {
	return nil
}
