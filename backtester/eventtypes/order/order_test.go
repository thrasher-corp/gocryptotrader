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
		Base: &event.Base{
			CurrencyPair: currency.NewBTCUSDT(),
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
		AllocatedFunds: decimal.NewFromInt(1337),
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
	if !k.GetBuyLimit().Equal(decimal.NewFromInt(1337)) {
		t.Errorf("received '%v' expected '%v'", k.GetBuyLimit(), decimal.NewFromInt(1337))
	}
}

func TestGetSellLimit(t *testing.T) {
	t.Parallel()
	k := Order{
		SellLimit: decimal.NewFromInt(1337),
	}
	if !k.GetSellLimit().Equal(decimal.NewFromInt(1337)) {
		t.Errorf("received '%v' expected '%v'", k.GetSellLimit(), decimal.NewFromInt(1337))
	}
}

func TestPair(t *testing.T) {
	t.Parallel()
	cp := currency.NewBTCUSDT()
	k := Order{
		Base: &event.Base{
			CurrencyPair: cp,
		},
	}
	if !k.Pair().Equal(cp) {
		t.Errorf("received '%v' expected '%v'", k.Pair(), cp)
	}
}

func TestGetStatus(t *testing.T) {
	t.Parallel()
	k := Order{
		Status: gctorder.UnknownStatus,
	}
	if k.GetStatus() != gctorder.UnknownStatus {
		t.Errorf("received '%v' expected '%v'", k.GetStatus(), gctorder.UnknownStatus)
	}
}

func TestGetFillDependentEvent(t *testing.T) {
	t.Parallel()
	k := Order{
		FillDependentEvent: &signal.Signal{Amount: decimal.NewFromInt(1337)},
	}
	if !k.GetFillDependentEvent().GetAmount().Equal(decimal.NewFromInt(1337)) {
		t.Errorf("received '%v' expected '%v'", k.GetFillDependentEvent(), decimal.NewFromInt(1337))
	}
}

func TestIsClosingPosition(t *testing.T) {
	t.Parallel()
	k := Order{
		ClosingPosition: true,
	}
	if !k.IsClosingPosition() {
		t.Errorf("received '%v' expected '%v'", k.IsClosingPosition(), true)
	}
}
