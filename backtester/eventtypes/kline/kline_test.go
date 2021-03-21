package kline

import (
	"testing"
)

func TestClose(t *testing.T) {
	k := Kline{
		Close: 1337,
	}
	if k.ClosePrice() != 1337 {
		t.Error("expected 1337")
	}
}

func TestHigh(t *testing.T) {
	k := Kline{
		High: 1337,
	}
	if k.HighPrice() != 1337 {
		t.Error("expected 1337")
	}
}

func TestLow(t *testing.T) {
	k := Kline{
		Low: 1337,
	}
	if k.LowPrice() != 1337 {
		t.Error("expected 1337")
	}
}

func TestOpen(t *testing.T) {
	k := Kline{
		Open: 1337,
	}
	if k.OpenPrice() != 1337 {
		t.Error("expected 1337")
	}
}
