package kline

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestClose(t *testing.T) {
	t.Parallel()
	k := Kline{
		Close: decimal.NewFromInt(1337),
	}
	if !k.ClosePrice().Equal(decimal.NewFromInt(1337)) {
		t.Error("expected decimal.NewFromInt(1337)")
	}
}

func TestHigh(t *testing.T) {
	t.Parallel()
	k := Kline{
		High: decimal.NewFromInt(1337),
	}
	if !k.HighPrice().Equal(decimal.NewFromInt(1337)) {
		t.Error("expected decimal.NewFromInt(1337)")
	}
}

func TestLow(t *testing.T) {
	t.Parallel()
	k := Kline{
		Low: decimal.NewFromInt(1337),
	}
	if !k.LowPrice().Equal(decimal.NewFromInt(1337)) {
		t.Error("expected decimal.NewFromInt(1337)")
	}
}

func TestOpen(t *testing.T) {
	t.Parallel()
	k := Kline{
		Open: decimal.NewFromInt(1337),
	}
	if !k.OpenPrice().Equal(decimal.NewFromInt(1337)) {
		t.Error("expected decimal.NewFromInt(1337)")
	}
}
