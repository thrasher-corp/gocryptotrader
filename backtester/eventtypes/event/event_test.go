package event

import (
	"strings"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

func TestGetConcatReasons(t *testing.T) {
	t.Parallel()
	e := &Base{}
	e.AppendReason("test")
	y := e.GetConcatReasons()
	if !strings.Contains(y, "test") {
		t.Error("expected test")
	}
	e.AppendReason("test")
	y = e.GetConcatReasons()
	if y != "test. test" {
		t.Error("expected 'test. test'")
	}
}

func TestGetReasons(t *testing.T) {
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

func TestGetAssetType(t *testing.T) {
	t.Parallel()
	e := &Base{
		AssetType: asset.Spot,
	}
	if y := e.GetAssetType(); y != asset.Spot {
		t.Error("expected spot")
	}
}

func TestGetExchange(t *testing.T) {
	t.Parallel()
	e := &Base{
		Exchange: "test",
	}
	if y := e.GetExchange(); y != "test" {
		t.Error("expected test")
	}
}

func TestGetInterval(t *testing.T) {
	t.Parallel()
	e := &Base{
		Interval: gctkline.OneMin,
	}
	if y := e.GetInterval(); y != gctkline.OneMin {
		t.Error("expected one minute")
	}
}

func TestGetTime(t *testing.T) {
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

func TestIsEvent(t *testing.T) {
	t.Parallel()
	e := &Base{}
	if y := e.IsEvent(); !y {
		t.Error("it is an event")
	}
}

func TestPair(t *testing.T) {
	t.Parallel()
	e := &Base{
		CurrencyPair: currency.NewBTCUSDT(),
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
	if b.GetConcatReasons() != "hello moto" {
		t.Errorf("expected hello moto, received '%v'", b.GetConcatReasons())
	}
	b.AppendReasonf("%v %v", "hello", "moto")
	if b.GetConcatReasons() != "hello moto. hello moto" {
		t.Errorf("expected 'hello moto. hello moto', received '%v'", b.GetConcatReasons())
	}
}

func TestGetBase(t *testing.T) {
	t.Parallel()
	b1 := &Base{
		Exchange: "hello",
	}
	if b1.Exchange != b1.GetBase().Exchange {
		t.Errorf("expected '%v' received '%v'", b1.Exchange, b1.GetBase().Exchange)
	}
}

func TestGetUnderlyingPair(t *testing.T) {
	t.Parallel()
	b1 := &Base{
		UnderlyingPair: currency.NewBTCUSDT(),
	}
	if !b1.UnderlyingPair.Equal(b1.GetUnderlyingPair()) {
		t.Errorf("expected '%v' received '%v'", b1.UnderlyingPair, b1.GetUnderlyingPair())
	}
}
