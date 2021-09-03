package kline

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestClose(t *testing.T) {
	k := Kline{
		Close: decimal.NewFromInt(1337),
	}
	if k.ClosePrice() != decimal.NewFromInt(1337) {
		t.Error("expected decimal.NewFromInt(1337)")
	}
}

func TestHigh(t *testing.T) {
	k := Kline{
		High: decimal.NewFromInt(1337),
	}
	if k.HighPrice() != decimal.NewFromInt(1337) {
		t.Error("expected decimal.NewFromInt(1337)")
	}
}

func TestLow(t *testing.T) {
	k := Kline{
		Low: decimal.NewFromInt(1337),
	}
	if k.LowPrice() != decimal.NewFromInt(1337) {
		t.Error("expected decimal.NewFromInt(1337)")
	}
}

func TestOpen(t *testing.T) {
	k := Kline{
		Open: decimal.NewFromInt(1337),
	}
	if k.OpenPrice() != decimal.NewFromInt(1337) {
		t.Error("expected decimal.NewFromInt(1337)")
	}
}
