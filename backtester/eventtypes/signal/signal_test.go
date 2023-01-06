package signal

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/kline"
	"github.com/thrasher-corp/gocryptotrader/currency"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func TestIsSignal(t *testing.T) {
	t.Parallel()
	s := Signal{}
	if !s.IsSignal() {
		t.Error("expected true")
	}
}

func TestSetDirection(t *testing.T) {
	t.Parallel()
	s := Signal{Direction: gctorder.Sell}
	s.SetDirection(gctorder.Buy)
	if s.GetDirection() != gctorder.Buy {
		t.Error("expected buy")
	}
}

func TestSetPrice(t *testing.T) {
	t.Parallel()
	s := Signal{
		ClosePrice: decimal.NewFromInt(1),
	}
	s.SetPrice(decimal.NewFromInt(1337))
	if !s.GetClosePrice().Equal(decimal.NewFromInt(1337)) {
		t.Errorf("received '%v' expected '%v'", s.GetClosePrice(), 1337)
	}
}

func TestSetBuyLimit(t *testing.T) {
	t.Parallel()
	s := Signal{
		BuyLimit: decimal.NewFromInt(10),
	}
	s.SetBuyLimit(decimal.NewFromInt(20))
	if !s.GetBuyLimit().Equal(decimal.NewFromInt(20)) {
		t.Errorf("expected 20, received %v", s.GetBuyLimit())
	}
}

func TestSetSellLimit(t *testing.T) {
	t.Parallel()
	s := Signal{
		SellLimit: decimal.NewFromInt(10),
	}
	s.SetSellLimit(decimal.NewFromInt(20))
	if !s.GetSellLimit().Equal(decimal.NewFromInt(20)) {
		t.Errorf("expected 20, received %v", s.GetSellLimit())
	}
}

func TestGetAmount(t *testing.T) {
	t.Parallel()
	s := Signal{
		Amount: decimal.NewFromInt(1337),
	}
	if !s.GetAmount().Equal(decimal.NewFromInt(1337)) {
		t.Error("expected decimal.NewFromInt(1337)")
	}
}

func TestSetAmount(t *testing.T) {
	t.Parallel()
	s := Signal{}
	s.SetAmount(decimal.NewFromInt(1337))
	if !s.GetAmount().Equal(decimal.NewFromInt(1337)) {
		t.Error("expected decimal.NewFromInt(1337)")
	}
}

func TestGetUnderlyingPair(t *testing.T) {
	t.Parallel()
	s := Signal{
		Base: &event.Base{
			UnderlyingPair: currency.NewPair(currency.USD, currency.DOGE),
		},
	}
	if !s.GetUnderlyingPair().Equal(s.Base.UnderlyingPair) {
		t.Errorf("expected '%v'", s.Base.UnderlyingPair)
	}
}

func TestPair(t *testing.T) {
	t.Parallel()
	s := Signal{
		Base: &event.Base{
			CurrencyPair: currency.NewPair(currency.USD, currency.DOGE),
		},
	}
	if !s.Pair().Equal(s.Base.CurrencyPair) {
		t.Errorf("expected '%v'", s.Base.CurrencyPair)
	}
}

func TestGetFillDependentEvent(t *testing.T) {
	t.Parallel()
	s := Signal{}
	if a := s.GetFillDependentEvent(); a != nil {
		t.Error("expected nil")
	}
	s.FillDependentEvent = &Signal{
		Amount: decimal.NewFromInt(1337),
	}
	e := s.GetFillDependentEvent()
	if !e.GetAmount().Equal(decimal.NewFromInt(1337)) {
		t.Error("expected 1337")
	}
}

func TestGetCollateralCurrency(t *testing.T) {
	t.Parallel()
	s := Signal{}
	c := s.GetCollateralCurrency()
	if !c.IsEmpty() {
		t.Error("expected empty currency")
	}
	s.CollateralCurrency = currency.BTC
	c = s.GetCollateralCurrency()
	if !c.Equal(currency.BTC) {
		t.Error("expected empty currency")
	}
}

func TestIsNil(t *testing.T) {
	t.Parallel()
	s := &Signal{}
	if s.IsNil() {
		t.Error("expected false")
	}
	s = nil
	if !s.IsNil() {
		t.Error("expected true")
	}
}

func TestMatchOrderAmount(t *testing.T) {
	t.Parallel()
	s := &Signal{}
	if s.MatchOrderAmount() {
		t.Error("expected false")
	}
	s.MatchesOrderAmount = true
	if !s.MatchOrderAmount() {
		t.Error("expected true")
	}
}

func TestGetHighPrice(t *testing.T) {
	t.Parallel()
	s := Signal{
		HighPrice: decimal.NewFromInt(1337),
	}
	if !s.GetHighPrice().Equal(decimal.NewFromInt(1337)) {
		t.Errorf("received '%v' expected '%v'", s.GetHighPrice(), 1337)
	}
}

func TestGetLowPrice(t *testing.T) {
	t.Parallel()
	s := Signal{
		LowPrice: decimal.NewFromInt(1337),
	}
	if !s.GetLowPrice().Equal(decimal.NewFromInt(1337)) {
		t.Errorf("received '%v' expected '%v'", s.GetLowPrice(), 1337)
	}
}

func TestGetOpenPrice(t *testing.T) {
	t.Parallel()
	s := Signal{
		OpenPrice: decimal.NewFromInt(1337),
	}
	if !s.GetOpenPrice().Equal(decimal.NewFromInt(1337)) {
		t.Errorf("received '%v' expected '%v'", s.GetOpenPrice(), 1337)
	}
}

func TestToKline(t *testing.T) {
	t.Parallel()
	s := Signal{
		OpenPrice: decimal.NewFromInt(1337),
	}
	k := s.ToKline()
	switch k.(type) {
	case kline.Event:
		if !k.GetOpenPrice().Equal(decimal.NewFromInt(1337)) {
			t.Errorf("received '%v' expected '%v'", k.GetOpenPrice(), 1337)
		}
	default:
		t.Errorf("expected  '%v' received '%v'", "kline event", "signal event")
	}
}
