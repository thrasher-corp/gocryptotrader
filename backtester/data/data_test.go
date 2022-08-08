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
	time int
}

type fakeHandler struct{}

func TestLatest(t *testing.T) {
	t.Parallel()
	var d Base
	d.AppendStream(&fakeEvent{time: 1})
	if latest := d.Latest(); latest != d.stream[d.offset] {
		t.Error("expected latest to match offset")
	}
}

func TestBaseDataFunctions(t *testing.T) {
	t.Parallel()
	var d Base

	d.Next()
	o := d.Offset()
	if o != 0 {
		t.Error("expected 0")
	}
	d.AppendStream(nil)
	if d.IsLastEvent() {
		t.Error("no")
	}
	d.AppendStream(nil)
	if len(d.stream) != 0 {
		t.Error("expected 0")
	}
	d.AppendStream(&fakeEvent{time: 1, Base: &event.Base{Offset: 1}})
	d.AppendStream(&fakeEvent{time: 2, Base: &event.Base{Offset: 2}})
	d.AppendStream(&fakeEvent{time: 3, Base: &event.Base{Offset: 3}})
	d.AppendStream(&fakeEvent{time: 4, Base: &event.Base{Offset: 4}})
	d.Next()

	d.Next()
	if list := d.List(); len(list) != 2 {
		t.Errorf("expected 2 received %v", len(list))
	}
	d.Next()
	d.Next()
	if !d.IsLastEvent() {
		t.Error("expected last event")
	}
	o = d.Offset()
	if o != 4 {
		t.Error("expected 4")
	}
	if list := d.List(); len(list) != 0 {
		t.Error("expected 0")
	}
	if history := d.History(); len(history) != 4 {
		t.Errorf("expected 4 received %v", len(history))
	}

	d.SetStream(nil)
	if st := d.GetStream(); st != nil {
		t.Error("expected nil")
	}
	d.Reset()
	d.GetStream()
	d.SortStream()
}

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
	if d.data[exch][a][p] != nil {
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
	if len(result[exch][a]) != 2 {
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
	d := HandlerPerCurrency{}
	d.SetDataForCurrency(exch, a, p, nil)
	d.SetDataForCurrency(exch, a, currency.NewPair(currency.BTC, currency.DOGE), nil)
	d.Reset()
	if d.data != nil {
		t.Error("expected nil")
	}
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

// methods that satisfy the common.Event interface
func (f fakeEvent) GetOffset() int64 {
	return f.Offset
}

func (f fakeEvent) SetOffset(int64) {
}

func (f fakeEvent) IsEvent() bool {
	return false
}

func (f fakeEvent) GetTime() time.Time {
	return time.Now().Add(time.Hour * time.Duration(f.time))
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

func (f fakeHandler) AppendStream(s ...Event) {
	return
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

func (f fakeHandler) Offset() int {
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
