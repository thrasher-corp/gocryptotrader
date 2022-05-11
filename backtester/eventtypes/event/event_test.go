package event

import (
	"strings"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

func TestEvent_GetReason(t *testing.T) {
	t.Parallel()
	e := &Base{}
	e.AppendReason("test")
	y := e.GetReason()
	if !strings.Contains(y, "test") {
		t.Error("expected test")
	}
	e.AppendReason("test")
	y = e.GetReason()
	if y != "test. test" {
		t.Error("expected 'test. test'")
	}
}

func TestEvent_GetReasons(t *testing.T) {
	t.Parallel()
	e := &Base{}
	e.AppendReason("test")
	y := e.GetReasons()
	if !strings.Contains(y[0], "test") {
		t.Error("expected test")
	}
	e.AppendReason("test2")
	y = e.GetReasons()
	if y[1] != "test2" {
		t.Error("expected 'test2'")
	}
}

func TestEvent_GetAssetType(t *testing.T) {
	t.Parallel()
	e := &Base{
		AssetType: asset.Spot,
	}
	if y := e.GetAssetType(); y != asset.Spot {
		t.Error("expected spot")
	}
}

func TestEvent_GetExchange(t *testing.T) {
	t.Parallel()
	e := &Base{
		Exchange: "test",
	}
	if y := e.GetExchange(); y != "test" {
		t.Error("expected test")
	}
}

func TestEvent_GetInterval(t *testing.T) {
	t.Parallel()
	e := &Base{
		Interval: gctkline.OneMin,
	}
	if y := e.GetInterval(); y != gctkline.OneMin {
		t.Error("expected one minute")
	}
}

func TestEvent_GetTime(t *testing.T) {
	t.Parallel()
	tt := time.Now()
	e := &Base{
		Time: tt,
	}
	y := e.GetTime()
	if !y.Equal(tt) {
		t.Errorf("expected %v", tt)
	}
}

func TestEvent_IsEvent(t *testing.T) {
	t.Parallel()
	e := &Base{}
	if y := e.IsEvent(); !y {
		t.Error("it is an event")
	}
}

func TestEvent_Pair(t *testing.T) {
	t.Parallel()
	e := &Base{
		CurrencyPair: currency.NewPair(currency.BTC, currency.USDT),
	}
	y := e.Pair()
	if y.IsEmpty() {
		t.Error("expected currency")
	}
}

func TestGetOffset(t *testing.T) {
	t.Parallel()
	b := Base{
		Offset: 1337,
	}
	if b.GetOffset() != 1337 {
		t.Error("expected 1337")
	}
}

func TestSetOffset(t *testing.T) {
	t.Parallel()
	b := Base{
		Offset: 1337,
	}
	b.SetOffset(1339)
	if b.Offset != 1339 {
		t.Error("expected 1339")
	}
}

func TestAppendReasonf(t *testing.T) {
	t.Parallel()
	b := Base{}
	b.AppendReasonf("%v", "hello moto")
	if b.Reason != "hello moto" {
		t.Errorf("epected hello moto, received '%v'", b.Reason)
	}
	b.AppendReasonf("%v %v", "hello", "moto")
	if b.Reason != "hello moto. hello moto" {
		t.Errorf("epected 'hello moto. hello moto', received '%v'", b.Reason)
	}
}
