package data

import (
	"testing"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

func TestSomething(t *testing.T) {
	// This test is more fully implemented in the fully implemented kline_test
	var d Data
	err := d.Load()
	if err != nil {
		t.Error(err)
	}
	d.Latest()
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
	d.List()
	d.History()
	d.SetStream(nil)
	st := d.GetStream()
	if st != nil {
		t.Error("expected nil")
	}
	d.Reset()
	d.GetStream()
	d.StreamOpen()
	d.StreamHigh()
	d.StreamClose()
	d.StreamLow()
	d.StreamVol()
	d.SortStream()
}

func TestSetup(t *testing.T) {
	d := HandlerPerCurrency{}
	d.Setup()
	if d.data == nil {
		t.Error("expected not nil")
	}
}

func TestSetDataForCurrency(t *testing.T) {
	d := HandlerPerCurrency{}
	exch := "binance"
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
	d := HandlerPerCurrency{}
	exch := "binance"
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
	d := HandlerPerCurrency{}
	exch := "binance"
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
	d := HandlerPerCurrency{}
	exch := "binance"
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	d.SetDataForCurrency(exch, a, p, nil)
	d.SetDataForCurrency(exch, a, currency.NewPair(currency.BTC, currency.DOGE), nil)
	d.Reset()
	if d.data != nil {
		t.Error("expected nil")
	}
}
