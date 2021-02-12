package order

import (
	"testing"

	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/currency"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func TestIsOrder(t *testing.T) {
	o := Order{}
	if !o.IsOrder() {
		t.Error("expected true")
	}
}

func TestSetDirection(t *testing.T) {
	o := Order{
		Direction: gctorder.Sell,
	}
	o.SetDirection(gctorder.Buy)
	if o.GetDirection() != gctorder.Buy {
		t.Error("expected buy")
	}
}

func TestSetAmount(t *testing.T) {
	o := Order{
		Amount: 1,
	}
	o.SetAmount(1337)
	if o.GetAmount() != 1337 {
		t.Error("expected 1337")
	}
}

func TestPair(t *testing.T) {
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
	o := Order{
		ID: "1337",
	}
	o.SetID("1338")
	if o.GetID() != "1338" {
		t.Error("expected 1338")
	}
}

func TestLeverage(t *testing.T) {
	o := Order{
		Leverage: 1,
	}
	o.SetLeverage(1337)
	if o.GetLeverage() != 1337 || !o.IsLeveraged() {
		t.Error("expected leverage")
	}
}

func TestGetFunds(t *testing.T) {
	o := Order{
		Funds: 1337,
	}
	funds := o.GetFunds()
	if funds != 1337 {
		t.Error("expected 1337")
	}
}
