package fill

import (
	"testing"

	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func TestSetDirection(t *testing.T) {
	f := Fill{
		Direction: gctorder.Sell,
	}
	f.SetDirection(gctorder.Buy)
	if f.GetDirection() != gctorder.Buy {
		t.Error("expected buy")
	}
}

func TestSetAmount(t *testing.T) {
	f := Fill{
		Amount: 1,
	}
	f.SetAmount(1337)
	if f.GetAmount() != 1337 {
		t.Error("expected 1337")
	}
}

func TestGetClosePrice(t *testing.T) {
	f := Fill{
		ClosePrice: 1337,
	}
	if f.GetClosePrice() != 1337 {
		t.Error("expected 1337")
	}
}

func TestGetVolumeAdjustedPrice(t *testing.T) {
	f := Fill{
		VolumeAdjustedPrice: 1337,
	}
	if f.GetVolumeAdjustedPrice() != 1337 {
		t.Error("expected 1337")
	}
}

func TestGetPurchasePrice(t *testing.T) {
	f := Fill{
		PurchasePrice: 1337,
	}
	if f.GetPurchasePrice() != 1337 {
		t.Error("expected 1337")
	}
}

func TestSetExchangeFee(t *testing.T) {
	f := Fill{
		ExchangeFee: 1,
	}
	f.SetExchangeFee(1337)
	if f.GetExchangeFee() != 1337 {
		t.Error("expected 1337")
	}
}

func TestGetOrder(t *testing.T) {
	f := Fill{
		Order: &gctorder.Detail{},
	}
	if f.GetOrder() == nil {
		t.Error("expected not nil")
	}
}

func TestGetSlippageRate(t *testing.T) {
	f := Fill{
		Slippage: 1,
	}
	if f.GetSlippageRate() != 1 {
		t.Error("expected 1")
	}
}
