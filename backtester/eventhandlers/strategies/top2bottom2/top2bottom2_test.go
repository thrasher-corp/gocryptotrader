package top2bottom2

import (
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/base"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	eventkline "github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
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
	if !s.SupportsSimultaneousProcessing() {
		t.Error("expected true")
	}
}

func TestDescription(t *testing.T) {
	t.Parallel()
	s := Strategy{}
	if s.Description() != description {
		t.Error("unexpected description")
	}
}

func TestSetCustomSettings(t *testing.T) {
	t.Parallel()
	s := Strategy{}
	err := s.SetCustomSettings(nil)
	assert.NoError(t, err)

	float14 := float64(14)
	mappalopalous := make(map[string]any)
	mappalopalous[mfiPeriodKey] = float14
	mappalopalous[mfiLowKey] = float14
	mappalopalous[mfiHighKey] = float14

	err = s.SetCustomSettings(mappalopalous)
	assert.NoError(t, err)

	mappalopalous[mfiPeriodKey] = "14"
	err = s.SetCustomSettings(mappalopalous)
	assert.ErrorIs(t, err, base.ErrInvalidCustomSettings)

	mappalopalous[mfiPeriodKey] = float14
	mappalopalous[mfiLowKey] = "14"
	err = s.SetCustomSettings(mappalopalous)
	assert.ErrorIs(t, err, base.ErrInvalidCustomSettings)

	mappalopalous[mfiLowKey] = float14
	mappalopalous[mfiHighKey] = "14"
	err = s.SetCustomSettings(mappalopalous)
	assert.ErrorIs(t, err, base.ErrInvalidCustomSettings)

	mappalopalous[mfiHighKey] = float14
	mappalopalous["lol"] = float14
	err = s.SetCustomSettings(mappalopalous)
	assert.ErrorIs(t, err, base.ErrInvalidCustomSettings)
}

func TestOnSignal(t *testing.T) {
	t.Parallel()
	s := Strategy{}
	_, err := s.OnSignal(nil, nil, nil)
	assert.ErrorIs(t, err, errStrategyOnlySupportsSimultaneousProcessing)
}

func TestOnSignals(t *testing.T) {
	t.Parallel()
	s := Strategy{}
	_, err := s.OnSignal(nil, nil, nil)
	assert.ErrorIs(t, err, errStrategyOnlySupportsSimultaneousProcessing)

	dInsert := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	exch := "binance"
	a := asset.Spot
	p := currency.NewBTCUSDT()
	d := &data.Base{}
	err = d.SetStream([]data.Event{&eventkline.Kline{
		Base: &event.Base{
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
	assert.NoError(t, err)

	_, err = d.Next()
	assert.NoError(t, err)

	da := &kline.DataFromKline{
		Item:        &gctkline.Item{},
		Base:        d,
		RangeHolder: &gctkline.IntervalRangeHolder{},
	}
	_, err = s.OnSimultaneousSignals([]data.Handler{da}, nil, nil)
	if !strings.Contains(err.Error(), errStrategyCurrencyRequirements.Error()) {
		// common.Errs type doesn't keep type
		t.Errorf("received: %v, expected: %v", err, errStrategyCurrencyRequirements)
	}

	_, err = s.OnSimultaneousSignals([]data.Handler{da, da, da, da}, nil, nil)
	if !strings.Contains(err.Error(), base.ErrTooMuchBadData.Error()) {
		// common.Errs type doesn't keep type
		t.Errorf("received: %v, expected: %v", err, base.ErrTooMuchBadData)
	}
}

func TestSetDefaults(t *testing.T) {
	t.Parallel()
	s := Strategy{}
	s.SetDefaults()
	if !s.mfiHigh.Equal(decimal.NewFromInt(70)) {
		t.Error("expected 70")
	}
	if !s.mfiLow.Equal(decimal.NewFromInt(30)) {
		t.Error("expected 30")
	}
	if !s.mfiPeriod.Equal(decimal.NewFromInt(14)) {
		t.Error("expected 14")
	}
}

func TestSelectTopAndBottomPerformers(t *testing.T) {
	t.Parallel()
	s := Strategy{}
	s.SetDefaults()
	_, err := s.selectTopAndBottomPerformers(nil, nil)
	assert.NoError(t, err)

	b := &event.Base{}
	fundEvents := []mfiFundEvent{
		{
			event: &signal.Signal{
				Base:       b,
				ClosePrice: decimal.NewFromInt(99),
				Direction:  order.DoNothing,
			},
			mfi: decimal.NewFromInt(99),
		},
		{
			event: &signal.Signal{
				Base:       b,
				ClosePrice: decimal.NewFromInt(98),
				Direction:  order.DoNothing,
			},
			mfi: decimal.NewFromInt(98),
		},
		{
			event: &signal.Signal{
				Base:       b,
				ClosePrice: decimal.NewFromInt(1),
				Direction:  order.DoNothing,
			},
			mfi: decimal.NewFromInt(1),
		},
		{
			event: &signal.Signal{
				Base:       b,
				ClosePrice: decimal.NewFromInt(2),
				Direction:  order.DoNothing,
			},
			mfi: decimal.NewFromInt(2),
		},
		{
			event: &signal.Signal{
				Base:       b,
				ClosePrice: decimal.NewFromInt(50),
				Direction:  order.DoNothing,
			},
			mfi: decimal.NewFromInt(50),
		},
	}
	resp, err := s.selectTopAndBottomPerformers(fundEvents, nil)
	assert.NoError(t, err)

	if len(resp) != 5 {
		t.Error("expected 5 events")
	}
	for i := range resp {
		switch resp[i].GetDirection() {
		case order.Buy:
			if !resp[i].GetClosePrice().Equal(decimal.NewFromInt(1)) && !resp[i].GetClosePrice().Equal(decimal.NewFromInt(2)) {
				t.Error("expected 1 or 2")
			}
		case order.Sell:
			if !resp[i].GetClosePrice().Equal(decimal.NewFromInt(99)) && !resp[i].GetClosePrice().Equal(decimal.NewFromInt(98)) {
				t.Error("expected 99 or 98")
			}
		case order.DoNothing:
			if !resp[i].GetClosePrice().Equal(decimal.NewFromInt(50)) {
				t.Error("expected 50")
			}
		}
	}
}
