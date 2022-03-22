package base

import (
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	datakline "github.com/thrasher-corp/gocryptotrader/backtester/data/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/kline"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

func TestGetBase(t *testing.T) {
	t.Parallel()
	s := Strategy{}
	_, err := s.GetBaseData(nil)
	if !errors.Is(err, common.ErrNilArguments) {
		t.Errorf("received: %v, expected: %v", err, common.ErrNilArguments)
	}

	_, err = s.GetBaseData(&datakline.DataFromKline{})
	if !errors.Is(err, common.ErrNilEvent) {
		t.Errorf("received: %v, expected: %v", err, common.ErrNilEvent)
	}
	tt := time.Now()
	exch := "binance"
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	d := data.Base{}
	d.SetStream([]common.DataEventHandler{&kline.Kline{
		Base: event.Base{
			Exchange:     exch,
			Time:         tt,
			Interval:     gctkline.OneDay,
			CurrencyPair: p,
			AssetType:    a,
		},
		Open:   decimal.NewFromInt(1337),
		Close:  decimal.NewFromInt(1337),
		Low:    decimal.NewFromInt(1337),
		High:   decimal.NewFromInt(1337),
		Volume: decimal.NewFromInt(1337),
	}})

	d.Next()
	_, err = s.GetBaseData(&datakline.DataFromKline{
		Item:        gctkline.Item{},
		Base:        d,
		RangeHolder: &gctkline.IntervalRangeHolder{},
	})
	if err != nil {
		t.Error(err)
	}
}

func TestSetSimultaneousProcessing(t *testing.T) {
	t.Parallel()
	s := Strategy{}
	is := s.UsingSimultaneousProcessing()
	if is {
		t.Error("expected false")
	}
	s.SetSimultaneousProcessing(true)
	is = s.UsingSimultaneousProcessing()
	if !is {
		t.Error("expected true")
	}
}

func TestUsingExchangeLevelFunding(t *testing.T) {
	t.Parallel()
	s := &Strategy{}
	if s.UsingExchangeLevelFunding() {
		t.Error("expected false")
	}
	s.usingExchangeLevelFunding = true
	if !s.UsingExchangeLevelFunding() {
		t.Error("expected true")
	}
}

func TestSetExchangeLevelFunding(t *testing.T) {
	t.Parallel()
	s := &Strategy{}
	s.SetExchangeLevelFunding(true)
	if !s.UsingExchangeLevelFunding() {
		t.Error("expected true")
	}
	if !s.UsingExchangeLevelFunding() {
		t.Error("expected true")
	}
}
