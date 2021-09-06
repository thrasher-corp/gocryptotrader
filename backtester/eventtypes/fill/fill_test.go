package fill

import (
	"testing"

	"github.com/shopspring/decimal"
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
	f.SetAmount(decimal.NewFromInt(1337)
	if f.GetAmount() != decimal.NewFromInt(1337) {
		t.Error("expected decimal.NewFromInt(1337)")
	}
}

func TestGetClosePrice(t *testing.T) {
	f := Fill{
		ClosePrice: decimal.NewFromInt(1337),
	}
	if f.GetClosePrice() != decimal.NewFromInt(1337) {
		t.Error("expected decimal.NewFromInt(1337)")
	}
}

func TestGetVolumeAdjustedPrice(t *testing.T) {
	f := Fill{
		VolumeAdjustedPrice: decimal.NewFromInt(1337),
	}
	if f.GetVolumeAdjustedPrice() != decimal.NewFromInt(1337) {
		t.Error("expected decimal.NewFromInt(1337)")
	}
}

func TestGetPurchasePrice(t *testing.T) {
	f := Fill{
		PurchasePrice: decimal.NewFromInt(1337),
	}
	if f.GetPurchasePrice() != decimal.NewFromInt(1337) {
		t.Error("expected decimal.NewFromInt(1337)")
	}
}

func TestSetExchangeFee(t *testing.T) {
	f := Fill{
		ExchangeFee: 1,
	}
	f.SetExchangeFee(decimal.NewFromInt(1337)
	if f.GetExchangeFee() != decimal.NewFromInt(1337) {
		t.Error("expected decimal.NewFromInt(1337)")
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
