package rsi

import (
	"errors"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/base"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	eventkline "github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

func TestName(t *testing.T) {
	d := Strategy{}
	n := d.Name()
	if n != Name {
		t.Errorf("expected %v", Name)
	}
}

func TestSupportsSimultaneousProcessing(t *testing.T) {
	s := Strategy{}
	if s.SupportsSimultaneousProcessing() {
		t.Error("expected false")
	}
}

func TestSetCustomSettings(t *testing.T) {
	s := Strategy{}
	err := s.SetCustomSettings(nil)
	if err != nil {
		t.Error(err)
	}
	float14 := float64(14)
	mappalopalous := make(map[string]interface{})
	mappalopalous[rsiPeriodKey] = float14
	mappalopalous[rsiLowKey] = float14
	mappalopalous[rsiHighKey] = float14

	err = s.SetCustomSettings(mappalopalous)
	if err != nil {
		t.Error(err)
	}

	mappalopalous[rsiPeriodKey] = "14"
	err = s.SetCustomSettings(mappalopalous)
	if !errors.Is(err, base.ErrInvalidCustomSettings) {
		t.Errorf("expected: %v, received %v", base.ErrInvalidCustomSettings, err)
	}

	mappalopalous[rsiPeriodKey] = float14
	mappalopalous[rsiLowKey] = "14"
	err = s.SetCustomSettings(mappalopalous)
	if !errors.Is(err, base.ErrInvalidCustomSettings) {
		t.Errorf("expected: %v, received %v", base.ErrInvalidCustomSettings, err)
	}

	mappalopalous[rsiLowKey] = float14
	mappalopalous[rsiHighKey] = "14"
	err = s.SetCustomSettings(mappalopalous)
	if !errors.Is(err, base.ErrInvalidCustomSettings) {
		t.Errorf("expected: %v, received %v", base.ErrInvalidCustomSettings, err)
	}

	mappalopalous[rsiHighKey] = float14
	mappalopalous["lol"] = float14
	err = s.SetCustomSettings(mappalopalous)
	if !errors.Is(err, base.ErrInvalidCustomSettings) {
		t.Errorf("expected: %v, received %v", base.ErrInvalidCustomSettings, err)
	}
}

func TestOnSignal(t *testing.T) {
	s := Strategy{}
	_, err := s.OnSignal(nil, nil)
	if !errors.Is(err, common.ErrNilEvent) {
		t.Errorf("expected: %v, received %v", common.ErrNilEvent, err)
	}
	dStart := time.Date(2020, 1, 0, 0, 0, 0, 0, time.UTC)
	dInsert := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	dEnd := time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC)
	exch := "binance"
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	d := data.Base{}
	d.SetStream([]common.DataEventHandler{&eventkline.Kline{
		Base: event.Base{
			Offset:       3,
			Exchange:     exch,
			Time:         dInsert,
			Interval:     gctkline.OneDay,
			CurrencyPair: p,
			AssetType:    a,
		},
		Open:   1337,
		Close:  1337,
		Low:    1337,
		High:   1337,
		Volume: 1337,
	}},
	)
	d.Next()
	da := &kline.DataFromKline{
		Item:  gctkline.Item{},
		Base:  d,
		Range: &gctkline.IntervalRangeHolder{},
	}
	var resp signal.Event
	_, err = s.OnSignal(da, nil)
	if !errors.Is(err, base.ErrTooMuchBadData) {
		t.Fatalf("expected: %v, received %v", base.ErrTooMuchBadData, err)
	}

	s.rsiPeriod = 1
	_, err = s.OnSignal(da, nil)
	if err != nil {
		t.Error(err)
	}

	da.Item = gctkline.Item{
		Exchange: exch,
		Pair:     p,
		Asset:    a,
		Interval: gctkline.OneDay,
		Candles: []gctkline.Candle{
			{
				Time:   dInsert,
				Open:   1337,
				High:   1337,
				Low:    1337,
				Close:  1337,
				Volume: 1337,
			},
		},
	}
	err = da.Load()
	if err != nil {
		t.Error(err)
	}

	ranger, err := gctkline.CalculateCandleDateRanges(dStart, dEnd, gctkline.OneDay, 100000)
	if err != nil {
		t.Error(err)
	}
	da.Range = ranger
	da.Range.SetHasDataFromCandles(da.Item.Candles)
	resp, err = s.OnSignal(da, nil)
	if err != nil {
		t.Error(err)
	}
	if resp.GetDirection() != common.DoNothing {
		t.Error("expected do nothing")
	}
}

func TestOnSignals(t *testing.T) {
	s := Strategy{}
	_, err := s.OnSignal(nil, nil)
	if !errors.Is(err, common.ErrNilEvent) {
		t.Errorf("expected: %v, received %v", common.ErrNilEvent, err)
	}
	dInsert := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	exch := "binance"
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	d := data.Base{}
	d.SetStream([]common.DataEventHandler{&eventkline.Kline{
		Base: event.Base{
			Exchange:     exch,
			Time:         dInsert,
			Interval:     gctkline.OneDay,
			CurrencyPair: p,
			AssetType:    a,
		},
		Open:   1337,
		Close:  1337,
		Low:    1337,
		High:   1337,
		Volume: 1337,
	}})
	d.Next()
	da := &kline.DataFromKline{
		Item:  gctkline.Item{},
		Base:  d,
		Range: &gctkline.IntervalRangeHolder{},
	}
	_, err = s.OnSimultaneousSignals([]data.Handler{da}, nil)
	if !errors.Is(err, base.ErrSimultaneousProcessingNotSupported) {
		t.Errorf("expected: %v, received %v", base.ErrSimultaneousProcessingNotSupported, err)
	}
}

func TestSetDefaults(t *testing.T) {
	s := Strategy{}
	s.SetDefaults()
	if s.rsiHigh != 70.0 {
		t.Error("expected 70")
	}
	if s.rsiLow != 30.0 {
		t.Error("expected 30")
	}
	if s.rsiPeriod != 14.0 {
		t.Error("expected 14")
	}
}
