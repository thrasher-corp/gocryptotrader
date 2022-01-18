package ftxquarterlyfutures

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
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
	t.Parallel()
	d := Strategy{}
	if n := d.Name(); n != Name {
		t.Errorf("expected %v", Name)
	}
}

func TestSupportsSimultaneousProcessing(t *testing.T) {
	t.Parallel()
	s := Strategy{}
	if s.SupportsSimultaneousProcessing() {
		t.Error("expected false")
	}
}

func TestSetCustomSettings(t *testing.T) {
	t.Parallel()
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
		t.Errorf("received: %v, expected: %v", err, base.ErrInvalidCustomSettings)
	}

	mappalopalous[rsiPeriodKey] = float14
	mappalopalous[rsiLowKey] = "14"
	err = s.SetCustomSettings(mappalopalous)
	if !errors.Is(err, base.ErrInvalidCustomSettings) {
		t.Errorf("received: %v, expected: %v", err, base.ErrInvalidCustomSettings)
	}

	mappalopalous[rsiLowKey] = float14
	mappalopalous[rsiHighKey] = "14"
	err = s.SetCustomSettings(mappalopalous)
	if !errors.Is(err, base.ErrInvalidCustomSettings) {
		t.Errorf("received: %v, expected: %v", err, base.ErrInvalidCustomSettings)
	}

	mappalopalous[rsiHighKey] = float14
	mappalopalous["lol"] = float14
	err = s.SetCustomSettings(mappalopalous)
	if !errors.Is(err, base.ErrInvalidCustomSettings) {
		t.Errorf("received: %v, expected: %v", err, base.ErrInvalidCustomSettings)
	}
}

func TestOnSignal(t *testing.T) {
	t.Parallel()
	s := Strategy{
		rsiPeriod: decimal.NewFromInt(14),
	}
	_, err := s.OnSignal(nil, nil, nil)
	if !errors.Is(err, common.ErrNilEvent) {
		t.Errorf("received: %v, expected: %v", err, common.ErrNilEvent)
	}
	dStart := time.Date(2020, 1, 0, 0, 0, 0, 0, time.UTC)
	dInsert := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	dEnd := time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC)
	exch := "ftx"
	a := asset.Futures
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
		Open:   decimal.NewFromInt(1337),
		Close:  decimal.NewFromInt(1337),
		Low:    decimal.NewFromInt(1337),
		High:   decimal.NewFromInt(1337),
		Volume: decimal.NewFromInt(1337),
	}},
	)
	d.Next()
	da := &kline.DataFromKline{
		Item:        gctkline.Item{},
		Base:        d,
		RangeHolder: &gctkline.IntervalRangeHolder{},
	}
	var resp signal.Event
	_, err = s.OnSignal(da, nil, nil)
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v, received %v", nil, err)
	}
	s.rsiPeriod = decimal.NewFromInt(1)
	_, err = s.OnSignal(da, nil, nil)
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
	da.RangeHolder = ranger
	da.RangeHolder.SetHasDataFromCandles(da.Item.Candles)
	resp, err = s.OnSignal(da, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetDirection() != common.DoNothing {
		t.Error("expected do nothing")
	}
}

func TestOnSignals(t *testing.T) {
	t.Parallel()
	s := Strategy{}
	_, err := s.OnSignal(nil, nil, nil)
	if !errors.Is(err, common.ErrNilEvent) {
		t.Errorf("received: %v, expected: %v", err, common.ErrNilEvent)
	}
	dInsert := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	exch := "ftx"
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
		Open:   decimal.NewFromInt(1337),
		Close:  decimal.NewFromInt(1337),
		Low:    decimal.NewFromInt(1337),
		High:   decimal.NewFromInt(1337),
		Volume: decimal.NewFromInt(1337),
	}})
	d.Next()
	da := &kline.DataFromKline{
		Item:        gctkline.Item{},
		Base:        d,
		RangeHolder: &gctkline.IntervalRangeHolder{},
	}
	_, err = s.OnSimultaneousSignals([]data.Handler{da}, nil, nil)
	if !strings.Contains(err.Error(), base.ErrSimultaneousProcessingNotSupported.Error()) {
		// common.Errs type doesn't keep type
		t.Errorf("received: %v, expected: %v", err, base.ErrSimultaneousProcessingNotSupported)
	}
}

func TestSetDefaults(t *testing.T) {
	t.Parallel()
	s := Strategy{}
	s.SetDefaults()
	if !s.rsiHigh.Equal(decimal.NewFromInt(70)) {
		t.Error("expected 70")
	}
	if !s.rsiLow.Equal(decimal.NewFromInt(30)) {
		t.Error("expected 30")
	}
	if !s.rsiPeriod.Equal(decimal.NewFromInt(14)) {
		t.Error("expected 14")
	}
}
