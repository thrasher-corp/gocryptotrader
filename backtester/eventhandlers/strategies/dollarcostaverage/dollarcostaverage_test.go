package dollarcostaverage

import (
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	eventkline "github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func TestName(t *testing.T) {
	d := Strategy{}
	n := d.Name()
	if n != Name {
		t.Errorf("expected %v", Name)
	}
}

func TestSupportsMultiCurrency(t *testing.T) {
	s := Strategy{}
	if !s.SupportsMultiCurrency() {
		t.Error("expected true")
	}
}

func TestSetCustomSettings(t *testing.T) {
	s := Strategy{}
	err := s.SetCustomSettings(nil)
	if err != nil && err.Error() != "unsupported" {
		t.Error(err)
	}
	if err == nil {
		t.Error("expected unsupported")
	}
}

func TestOnSignal(t *testing.T) {
	s := Strategy{}
	_, err := s.OnSignal(nil, nil)
	if err != nil && err.Error() != "received nil data" {
		t.Error(err)
	}
	dStart := time.Date(2020, 1, 0, 0, 0, 0, 0, time.UTC)
	dInsert := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	dEnd := time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC)
	exch := "binance"
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	d := data.Data{}
	d.SetStream([]common.DataEventHandler{&eventkline.Kline{
		Event: event.Event{
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
		Data:  d,
		Range: gctkline.IntervalRangeHolder{},
	}
	var resp signal.Event
	resp, err = s.OnSignal(da, nil)
	if err != nil {
		t.Error(err)
	}
	if resp.GetDirection() != common.MissingData {
		t.Error("expected missing data")
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

	ranger := gctkline.CalculateCandleDateRanges(dStart, dEnd, gctkline.OneDay, 100000)
	da.Range = ranger
	_ = da.Range.Verify(da.Item.Candles)
	resp, err = s.OnSignal(da, nil)
	if err != nil {
		t.Error(err)
	}
	if resp.GetDirection() != gctorder.Buy {
		t.Error("expected buy")
	}
}

func TestOnSignals(t *testing.T) {
	s := Strategy{}
	_, err := s.OnSignal(nil, nil)
	if err != nil && err.Error() != "received nil data" {
		t.Error(err)
	}
	dStart := time.Date(2020, 1, 0, 0, 0, 0, 0, time.UTC)
	dInsert := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	dEnd := time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC)
	exch := "binance"
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	d := data.Data{}
	d.SetStream([]common.DataEventHandler{&eventkline.Kline{
		Event: event.Event{
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
		Data:  d,
		Range: gctkline.IntervalRangeHolder{},
	}
	var resp []signal.Event
	resp, err = s.OnSignals([]data.Handler{da}, nil)
	if err != nil {
		t.Error(err)
	}
	if len(resp) != 1 {
		t.Fatal("expected 1 response")
	}
	if resp[0].GetDirection() != common.MissingData {
		t.Error("expected missing data")
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

	ranger := gctkline.CalculateCandleDateRanges(dStart, dEnd, gctkline.OneDay, 100000)
	da.Range = ranger
	_ = da.Range.Verify(da.Item.Candles)
	resp, err = s.OnSignals([]data.Handler{da}, nil)
	if err != nil {
		t.Error(err)
	}
	if len(resp) != 1 {
		t.Fatal("expected 1 response")
	}
	if resp[0].GetDirection() != gctorder.Buy {
		t.Error("expected buy")
	}
}

func TestSetDefaults(t *testing.T) {
	s := Strategy{}
	s.SetDefaults()
	if s != (Strategy{}) {
		t.Error("expected no changes")
	}
}
