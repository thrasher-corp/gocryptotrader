package dollarcostaverage

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
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
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func TestName(t *testing.T) {
	d := Strategy{}
	if n := d.Name(); n != Name {
		t.Errorf("expected %v", Name)
	}
}

func TestSupportsSimultaneousProcessing(t *testing.T) {
	s := Strategy{}
	if !s.SupportsSimultaneousProcessing() {
		t.Error("expected true")
	}
}

func TestSetCustomSettings(t *testing.T) {
	s := Strategy{}
	err := s.SetCustomSettings(nil)
	assert.ErrorIs(t, err, base.ErrCustomSettingsUnsupported)
}

func TestOnSignal(t *testing.T) {
	s := Strategy{}
	_, err := s.OnSignal(nil, nil, nil)
	assert.ErrorIs(t, err, common.ErrNilEvent)

	dStart := time.Date(2020, 1, 0, 0, 0, 0, 0, time.UTC)
	dEnd := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	exch := "binance"
	a := asset.Spot
	p := currency.NewBTCUSDT()
	d := &data.Base{}
	err = d.SetStream([]data.Event{&eventkline.Kline{
		Base: &event.Base{
			Exchange:     exch,
			Time:         dStart,
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
	assert.NoError(t, err)

	_, err = d.Next()
	assert.NoError(t, err)

	da := &kline.DataFromKline{
		Item:        &gctkline.Item{},
		Base:        d,
		RangeHolder: &gctkline.IntervalRangeHolder{},
	}
	var resp signal.Event
	resp, err = s.OnSignal(da, nil, nil)
	assert.NoError(t, err)

	if resp.GetDirection() != gctorder.MissingData {
		t.Error("expected missing data")
	}

	da.Item = &gctkline.Item{
		Exchange: exch,
		Pair:     p,
		Asset:    a,
		Interval: gctkline.OneDay,
		Candles: []gctkline.Candle{
			{
				Time:   dStart,
				Open:   1337,
				High:   1337,
				Low:    1337,
				Close:  1337,
				Volume: 1337,
			},
		},
	}
	err = da.Load()
	assert.NoError(t, err)

	ranger, err := gctkline.CalculateCandleDateRanges(dStart, dEnd, gctkline.OneDay, 100000)
	assert.NoError(t, err)

	da.RangeHolder = ranger
	err = da.RangeHolder.SetHasDataFromCandles(da.Item.Candles)
	assert.NoError(t, err)

	resp, err = s.OnSignal(da, nil, nil)
	assert.NoError(t, err)

	if resp.GetDirection() != gctorder.Buy {
		t.Errorf("expected buy, received %v", resp.GetDirection())
	}
}

func TestOnSignals(t *testing.T) {
	s := Strategy{}
	_, err := s.OnSignal(nil, nil, nil)
	assert.ErrorIs(t, err, common.ErrNilEvent)

	dStart := time.Date(2020, 1, 0, 0, 0, 0, 0, time.UTC)
	dEnd := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	exch := "binance"
	a := asset.Spot
	p := currency.NewBTCUSDT()
	d := &data.Base{}
	err = d.SetStream([]data.Event{&eventkline.Kline{
		Base: &event.Base{
			Offset:       1,
			Exchange:     exch,
			Time:         dStart,
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
	assert.NoError(t, err)

	_, err = d.Next()
	assert.NoError(t, err)

	da := &kline.DataFromKline{
		Item:        &gctkline.Item{},
		Base:        d,
		RangeHolder: &gctkline.IntervalRangeHolder{},
	}
	var resp []signal.Event
	resp, err = s.OnSimultaneousSignals([]data.Handler{da}, nil, nil)
	assert.NoError(t, err)

	if len(resp) != 1 {
		t.Fatal("expected 1 response")
	}
	if resp[0].GetDirection() != gctorder.MissingData {
		t.Error("expected missing data")
	}

	da.Item = &gctkline.Item{
		Exchange: exch,
		Pair:     p,
		Asset:    a,
		Interval: gctkline.OneDay,
		Candles: []gctkline.Candle{
			{
				Time:   dStart,
				Open:   1337,
				High:   1337,
				Low:    1337,
				Close:  1337,
				Volume: 1337,
			},
		},
	}
	err = da.Load()
	assert.NoError(t, err)

	ranger, err := gctkline.CalculateCandleDateRanges(dStart, dEnd, gctkline.OneDay, 100000)
	assert.NoError(t, err)

	da.RangeHolder = ranger
	err = da.RangeHolder.SetHasDataFromCandles(da.Item.Candles)
	assert.NoError(t, err)

	resp, err = s.OnSimultaneousSignals([]data.Handler{da}, nil, nil)
	assert.NoError(t, err)

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
