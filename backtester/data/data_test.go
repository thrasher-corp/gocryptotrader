package data

import (
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

const testExchange = "binance"

type fakeDataHandler struct {
	time int
}

func TestBaseDataFunctions(t *testing.T) {
	t.Parallel()
	var d Base
	latest := d.Latest()
	if latest != nil {
		t.Error("expected nil")
	}
	d.Next()
	o := d.Offset()
	if o != 0 {
		t.Error("expected 0")
	}
	d.AppendStream(nil)
	d.AppendStream(nil)
	d.AppendStream(nil)

	d.Next()
	o = d.Offset()
	if o != 0 {
		t.Error("expected 0")
	}
	list := d.List()
	if list != nil {
		t.Error("expected nil")
	}
	history := d.History()
	if history != nil {
		t.Error("expected nil")
	}
	d.SetStream(nil)
	st := d.GetStream()
	if st != nil {
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

func TestStream(t *testing.T) {
	var d Base
	var f fakeDataHandler

	// shut up coverage report
	f.GetOffset()
	f.SetOffset(1)
	f.IsEvent()
	f.Pair()
	f.GetExchange()
	f.GetInterval()
	f.GetAssetType()
	f.GetReason()
	f.AppendReason("fake")
	f.ClosePrice()
	f.HighPrice()
	f.LowPrice()
	f.OpenPrice()

	d.AppendStream(fakeDataHandler{time: 1})
	d.AppendStream(fakeDataHandler{time: 4})
	d.AppendStream(fakeDataHandler{time: 10})
	d.AppendStream(fakeDataHandler{time: 2})
	d.AppendStream(fakeDataHandler{time: 20})

	d.SortStream()

	f = d.Next().(fakeDataHandler)
	if f.time != 1 {
		t.Error("expected 1")
	}
	f = d.Next().(fakeDataHandler)
	if f.time != 2 {
		t.Error("expected 2")
	}
	f = d.Next().(fakeDataHandler)
	if f.time != 4 {
		t.Error("expected 4")
	}
	f = d.Next().(fakeDataHandler)
	if f.time != 10 {
		t.Error("expected 10")
	}
	f = d.Next().(fakeDataHandler)
	if f.time != 20 {
		t.Error("expected 20")
	}
}

func TestSetDataForCurrency(t *testing.T) {
	t.Parallel()
	d := HandlerPerCurrency{}
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
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
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
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
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	d.SetDataForCurrency(exch, a, p, nil)
	d.SetDataForCurrency(exch, a, currency.NewPair(currency.BTC, currency.DOGE), nil)
	result := d.GetDataForCurrency(exch, a, p)
	if result != nil {
		t.Error("expected nil")
	}
}

func TestReset(t *testing.T) {
	t.Parallel()
	d := HandlerPerCurrency{}
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	d.SetDataForCurrency(exch, a, p, nil)
	d.SetDataForCurrency(exch, a, currency.NewPair(currency.BTC, currency.DOGE), nil)
	d.Reset()
	if d.data != nil {
		t.Error("expected nil")
	}
}

// methods that satisfy the common.DataEventHandler interface
func (t fakeDataHandler) GetOffset() int64 {
	return 0
}

func (t fakeDataHandler) SetOffset(int64) {
}

func (t fakeDataHandler) IsEvent() bool {
	return false
}

func (t fakeDataHandler) GetTime() time.Time {
	return time.Now().Add(time.Hour * time.Duration(t.time))
}

func (t fakeDataHandler) Pair() currency.Pair {
	return currency.NewPair(currency.BTC, currency.USD)
}

func (t fakeDataHandler) GetExchange() string {
	return "fake"
}

func (t fakeDataHandler) GetInterval() kline.Interval {
	return kline.Interval(time.Minute)
}

func (t fakeDataHandler) GetAssetType() asset.Item {
	return asset.Spot
}

func (t fakeDataHandler) GetReason() string {
	return "fake"
}

func (t fakeDataHandler) AppendReason(string) {
}

func (t fakeDataHandler) ClosePrice() float64 {
	return 0
}

func (t fakeDataHandler) HighPrice() float64 {
	return 0
}

func (t fakeDataHandler) LowPrice() float64 {
	return 0
}

func (t fakeDataHandler) OpenPrice() float64 {
	return 0
}
