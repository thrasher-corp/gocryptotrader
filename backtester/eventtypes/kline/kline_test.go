package kline

import (
	"testing"
)

func TestClose(t *testing.T) {
	k := Kline{
		Close: 1337,
	}
	if k.Price() != 1337 {
		t.Error("expected 1337")
	}
}
