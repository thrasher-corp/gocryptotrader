package order

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
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

func TestPair(t *testing.T) {
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
		AllocatedFunds: decimal.NewFromInt(1337),
	}
	funds := o.GetAllocatedFunds()
	if !funds.Equal(decimal.NewFromInt(1337)) {
		t.Error("expected decimal.NewFromInt(1337)")
	}
}
