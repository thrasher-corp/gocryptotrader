package order

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func TestIsOrder(t *testing.T) {
	t.Parallel()
	o := Order{}
	if !o.IsOrder() {
		t.Error("expected true")
	}
}

func TestSetDirection(t *testing.T) {
	t.Parallel()
	o := Order{
		Direction: gctorder.Sell,
	}
	o.SetDirection(gctorder.Buy)
	if o.GetDirection() != gctorder.Buy {
		t.Error("expected buy")
	}
}

func TestSetAmount(t *testing.T) {
	t.Parallel()
	o := Order{
		Amount: decimal.NewFromInt(1),
	}
	o.SetAmount(decimal.NewFromInt(1337))
	if !o.GetAmount().Equal(decimal.NewFromInt(1337)) {
		t.Error("expected decimal.NewFromInt(1337)")
	}
}

func TestIsEmpty(t *testing.T) {
	t.Parallel()
	o := Order{
		Base: event.Base{
			CurrencyPair: currency.NewPair(currency.BTC, currency.USDT),
		},
	}
	y := o.CurrencyPair
	if y.IsEmpty() {
		t.Error("expected btc-usdt")
	}
}

func TestSetID(t *testing.T) {
	t.Parallel()
	o := Order{
		ID: "decimal.NewFromInt(1337)",
	}
	o.SetID("1338")
	if o.GetID() != "1338" {
		t.Error("expected 1338")
	}
}

func TestLeverage(t *testing.T) {
	t.Parallel()
	o := Order{
		Leverage: decimal.NewFromInt(1),
	}
	o.SetLeverage(decimal.NewFromInt(1337))
	if !o.GetLeverage().Equal(decimal.NewFromInt(1337)) || !o.IsLeveraged() {
		t.Error("expected leverage")
	}
}

func TestGetFunds(t *testing.T) {
	t.Parallel()
	o := Order{
		AllocatedSize: decimal.NewFromInt(1337),
	}
	funds := o.GetAllocatedFunds()
	if !funds.Equal(decimal.NewFromInt(1337)) {
		t.Error("expected decimal.NewFromInt(1337)")
	}
}

func TestOpen(t *testing.T) {
	t.Parallel()
	k := Order{
		ClosePrice: decimal.NewFromInt(1337),
	}
	if !k.GetClosePrice().Equal(decimal.NewFromInt(1337)) {
		t.Error("expected decimal.NewFromInt(1337)")
	}
}

func TestIsLiquidating(t *testing.T) {
	t.Parallel()
	k := Order{}
	if k.IsLiquidating() {
		t.Error("expected false")
	}
	k.LiquidatingPosition = true
	if !k.IsLiquidating() {
		t.Error("expected true")
	}
}

func TestGetBuyLimit(t *testing.T) {
	t.Parallel()
	k := Order{
		BuyLimit: decimal.NewFromInt(1337),
	}
	bl := k.GetBuyLimit()
	if !bl.Equal(decimal.NewFromInt(1337)) {
		t.Errorf("received '%v' expected '%v'", bl, decimal.NewFromInt(1337))
	}
}

func TestGetSellLimit(t *testing.T) {
	t.Parallel()
	k := Order{
		SellLimit: decimal.NewFromInt(1337),
	}
	sl := k.GetSellLimit()
	if !sl.Equal(decimal.NewFromInt(1337)) {
		t.Errorf("received '%v' expected '%v'", sl, decimal.NewFromInt(1337))
	}
}

func TestPair(t *testing.T) {
	t.Parallel()
	cp := currency.NewPair(currency.BTC, currency.USDT)
	k := Order{
		Base: event.Base{
			CurrencyPair: cp,
		},
	}
	p := k.Pair()
	if !p.Equal(cp) {
		t.Errorf("received '%v' expected '%v'", p, cp)
	}
}

func TestGetStatus(t *testing.T) {
	t.Parallel()
	k := Order{
		Status: gctorder.UnknownStatus,
	}
	s := k.GetStatus()
	if s != gctorder.UnknownStatus {
		t.Errorf("received '%v' expected '%v'", s, gctorder.UnknownStatus)
	}
}

func TestGetFillDependentEvent(t *testing.T) {
	t.Parallel()
	k := Order{
		FillDependentEvent: &signal.Signal{Amount: decimal.NewFromInt(1337)},
	}
	fde := k.GetFillDependentEvent()
	if !fde.GetAmount().Equal(decimal.NewFromInt(1337)) {
		t.Errorf("received '%v' expected '%v'", fde, decimal.NewFromInt(1337))
	}
}
func TestIsClosingPosition(t *testing.T) {
	t.Parallel()
	k := Order{
		ClosingPosition: true,
	}
	s := k.IsClosingPosition()
	if !s {
		t.Errorf("received '%v' expected '%v'", s, true)
	}
}
