package data

import (
	"errors"
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
	d.Reset()
	if d.data != nil {
		t.Error("expected nil")
	}
	d = nil
	d.Reset()
}

func TestBaseReset(t *testing.T) {
	t.Parallel()
	hello := &Base{offset: 1}
	hello.Reset()
	if hello.offset != 0 {
		t.Errorf("received '%v' expected '%v'", hello.offset, 0)
	}
	hello = nil
	hello.Reset()
}

func TestSortStream(t *testing.T) {
	t.Parallel()
	hello := Base{
		stream: []Event{
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
		},
	}
	hello.SortStream()
	if hello.stream[0].GetOffset() != 1337 {
		t.Error("expected 1337")
	}
}

func TestGetStream(t *testing.T) {
	t.Parallel()
	hello := Base{}
	resp := hello.GetStream()
	if len(resp) != 0 {
		t.Errorf("received '%v' expected '%v'", len(resp), 0)
	}
	hello.stream = []Event{
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
	resp = hello.GetStream()
	if len(resp) != 2 {
		t.Errorf("received '%v' expected '%v'", len(resp), 2)
	}
}

func TestOffset(t *testing.T) {
	t.Parallel()
	hello := Base{}
	o := hello.Offset()
	if o != 0 {
		t.Errorf("received '%v' expected '%v'", o, 0)
	}
	hello.offset = 1337
	o = hello.Offset()
	if o != 1337 {
		t.Errorf("received '%v' expected '%v'", o, 1337)
	}
}

func TestSetStream(t *testing.T) {
	t.Parallel()
	hello := Base{}
	hello.SetStream(nil)
	if len(hello.stream) != 0 {
		t.Errorf("received '%v' expected '%v'", len(hello.stream), 0)
	}
	hello.SetStream([]Event{
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
	if len(hello.stream) != 2 {
		t.Fatalf("received '%v' expected '%v'", len(hello.stream), 2)
	}
	if hello.stream[0].GetOffset() != 1 {
		t.Errorf("received '%v' expected '%v'", hello.stream[0].GetOffset(), 1)
	}
}

func TestNext(t *testing.T) {
	hello := Base{}
	hello.SetStream([]Event{
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
	resp := hello.Next()
	if resp != hello.stream[0] {
		t.Errorf("received '%v' expected '%v'", resp, hello.stream[0])
	}
	if hello.offset != 1 {
		t.Errorf("received '%v' expected '%v'", hello.offset, 1)
	}
	_ = hello.Next()
	resp = hello.Next()
	if resp != nil {
		t.Errorf("received '%v' expected '%v'", resp, nil)
	}
}

func TestHistory(t *testing.T) {
	hello := Base{}
	hello.SetStream([]Event{
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
	resp := hello.History()
	if len(resp) != 0 {
		t.Errorf("received '%v' expected '%v'", len(resp), 0)
	}

	_ = hello.Next()
	resp = hello.History()
	if len(resp) != 1 {
		t.Errorf("received '%v' expected '%v'", len(resp), 1)
	}
}

func TestLatest(t *testing.T) {
	hello := Base{}
	hello.SetStream([]Event{
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
	resp := hello.Latest()
	if resp != hello.stream[0] {
		t.Errorf("received '%v' expected '%v'", resp, hello.stream[0])
	}
	_ = hello.Next()
	resp = hello.Latest()
	if resp != hello.stream[0] {
		t.Errorf("received '%v' expected '%v'", resp, hello.stream[0])
	}

	_ = hello.Next()
	resp = hello.Latest()
	if resp != hello.stream[1] {
		t.Errorf("received '%v' expected '%v'", resp, hello.stream[1])
	}
}

func TestList(t *testing.T) {
	hello := Base{}
	hello.SetStream([]Event{
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
	list := hello.List()
	if len(list) != 2 {
		t.Errorf("received '%v' expected '%v'", len(list), 2)
	}
}

func TestIsLastEvent(t *testing.T) {
	t.Parallel()
	hello := Base{}
	hello.SetStream([]Event{
		&fakeEvent{
			Base: &event.Base{
				Offset: 1,
				Time:   time.Now(),
			},
		},
	})
	hello.latest = hello.stream[0]
	hello.offset = hello.stream[0].GetOffset()
	if !hello.IsLastEvent() {
		t.Errorf("received '%v' expected '%v'", false, true)
	}

	hello.isLiveData = true
	if hello.IsLastEvent() {
		t.Errorf("received '%v' expected '%v'", true, false)
	}
}

func TestIsLive(t *testing.T) {
	t.Parallel()
	hello := Base{}
	if hello.IsLive() {
		t.Error("expected false")
	}
	hello.isLiveData = true
	if !hello.IsLive() {
		t.Error("expected true")
	}
}

func TestSetLive(t *testing.T) {
	t.Parallel()
	hello := Base{}
	hello.SetLive(true)
	if !hello.isLiveData {
		t.Error("expected true")
	}

	hello.SetLive(false)
	if hello.isLiveData {
		t.Error("expected false")
	}
}

func TestAppendResults(t *testing.T) {
	t.Parallel()
	hello := Base{}
	validEvent := &fakeEvent{
		Base: &event.Base{},
	}
	hello.AppendStream(validEvent)
	if len(hello.stream) != 0 {
		t.Errorf("received '%v' expected '%v'", len(hello.stream), 0)
	}
	tt := time.Now()
	validEvent.Exchange = "hello"
	validEvent.AssetType = asset.Spot
	validEvent.CurrencyPair = currency.NewPair(currency.BTC, currency.USD)
	validEvent.Time = tt
	hello.AppendStream(validEvent)
	if len(hello.stream) != 1 {
		t.Errorf("received '%v' expected '%v'", len(hello.stream), 1)
	}

	hello.AppendStream(validEvent)
	if len(hello.stream) != 1 {
		t.Errorf("received '%v' expected '%v'", len(hello.stream), 1)
	}

	misMatchEvent := &fakeEvent{
		Base: &event.Base{
			Exchange:     "mismatch",
			CurrencyPair: currency.NewPair(currency.BTC, currency.DOGE),
			AssetType:    asset.Futures,
			Time:         tt,
		},
	}
	hello.AppendStream(misMatchEvent)
	if len(hello.stream) != 1 {
		t.Errorf("received '%v' expected '%v'", len(hello.stream), 1)
	}

	hello.AppendStream(nil)
	if len(hello.stream) != 1 {
		t.Errorf("received '%v' expected '%v'", len(hello.stream), 1)
	}
}

func TestEqualSource(t *testing.T) {
	t.Parallel()
	hello := Base{}

	if hello.equalSource(nil) {
		t.Errorf("received '%v' expected '%v'", false, true)
	}

	emptyEvent := &fakeEvent{
		Base: &event.Base{},
	}
	if hello.equalSource(emptyEvent) {
		t.Errorf("received '%v' expected '%v'", false, true)
	}

	validEvent := &fakeEvent{
		Base: &event.Base{
			Exchange:     "hello",
			CurrencyPair: currency.NewPair(currency.BTC, currency.USD),
			AssetType:    asset.Spot,
		},
	}
	if !hello.equalSource(validEvent) {
		t.Errorf("received '%v' expected '%v'", true, false)
	}

	hello.stream = append(hello.stream, validEvent)
	if !hello.equalSource(validEvent) {
		t.Errorf("received '%v' expected '%v'", true, false)
	}

	misMatchEvent := &fakeEvent{
		Base: &event.Base{
			Exchange:     "mismatch",
			CurrencyPair: currency.NewPair(currency.BTC, currency.DOGE),
			AssetType:    asset.Futures,
		},
	}
	if hello.equalSource(misMatchEvent) {
		t.Errorf("received '%v' expected '%v'", false, true)
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

func (f fakeHandler) AppendStream(s ...Event) {}

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

func (f fakeHandler) Reset() {}
