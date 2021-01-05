package kline

import (
	"testing"

	"github.com/thrasher-corp/gocryptotrader/backtester/data"
)

func TestDataType(t *testing.T) {
	k := Kline{}
	if k.DataType() != data.CandleType {
		t.Error("expected candletype")
	}
}

func TestClose(t *testing.T) {
	k := Kline{
		Close: 1337,
	}
	if k.Price() != 1337 {
		t.Error("expected 1337")
	}
}
