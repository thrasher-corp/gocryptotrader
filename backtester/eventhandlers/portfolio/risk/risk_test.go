package risk

import (
	"errors"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/compliance"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func TestAssessHoldingsRatio(t *testing.T) {
	t.Parallel()
	ratio := assessHoldingsRatio(currency.NewPair(currency.BTC, currency.USDT), []holdings.Holding{
		{
			Pair:      currency.NewPair(currency.BTC, currency.USDT),
			BaseValue: decimal.NewFromInt(2),
		},
		{
			Pair:      currency.NewPair(currency.LTC, currency.USDT),
			BaseValue: decimal.NewFromInt(2),
		},
	})
	if !ratio.Equal(decimal.NewFromFloat(0.5)) {
		t.Errorf("expected %v received %v", 0.5, ratio)
	}

	ratio = assessHoldingsRatio(currency.NewPair(currency.BTC, currency.USDT), []holdings.Holding{
		{
			Pair:      currency.NewPair(currency.BTC, currency.USDT),
			BaseValue: decimal.NewFromInt(1),
		},
		{
			Pair:      currency.NewPair(currency.LTC, currency.USDT),
			BaseValue: decimal.NewFromInt(2),
		},
		{
			Pair:      currency.NewPair(currency.DOGE, currency.USDT),
			BaseValue: decimal.NewFromInt(1),
		},
	})
	if !ratio.Equal(decimal.NewFromFloat(0.25)) {
		t.Errorf("expected %v received %v", 0.25, ratio)
	}
}

func TestEvaluateOrder(t *testing.T) {
	t.Parallel()
	r := Risk{}
	_, err := r.EvaluateOrder(nil, nil, compliance.Snapshot{})
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Error(err)
	}
	p := currency.NewPair(currency.BTC, currency.USDT)
	e := "binance"
	a := asset.Spot
	o := &order.Order{
		Base: &event.Base{
			Exchange:     e,
			AssetType:    a,
			CurrencyPair: p,
		},
	}
	h := []holdings.Holding{}
	r.CurrencySettings = make(map[string]map[asset.Item]map[*currency.Item]map[*currency.Item]*CurrencySettings)
	r.CurrencySettings[e] = make(map[asset.Item]map[*currency.Item]map[*currency.Item]*CurrencySettings)
	r.CurrencySettings[e][a] = make(map[*currency.Item]map[*currency.Item]*CurrencySettings)
	r.CurrencySettings[e][a][p.Base.Item] = make(map[*currency.Item]*CurrencySettings)
	_, err = r.EvaluateOrder(o, h, compliance.Snapshot{})
	if !errors.Is(err, errNoCurrencySettings) {
		t.Error(err)
	}

	r.CurrencySettings[e][a][p.Base.Item][p.Quote.Item] = &CurrencySettings{
		MaximumOrdersWithLeverageRatio: decimal.NewFromFloat(0.3),
		MaxLeverageRate:                decimal.NewFromFloat(0.3),
		MaximumHoldingRatio:            decimal.NewFromFloat(0.3),
	}

	h = append(h, holdings.Holding{
		Pair:     p,
		BaseSize: decimal.NewFromInt(1),
	})
	_, err = r.EvaluateOrder(o, h, compliance.Snapshot{})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}

	h = append(h, holdings.Holding{
		Pair: currency.NewPair(currency.DOGE, currency.USDT),
	})
	o.Leverage = decimal.NewFromFloat(1.1)
	r.CurrencySettings[e][a][p.Base.Item][p.Quote.Item].MaximumHoldingRatio = decimal.Zero
	_, err = r.EvaluateOrder(o, h, compliance.Snapshot{})
	if !errors.Is(err, errLeverageNotAllowed) {
		t.Error(err)
	}
	r.CanUseLeverage = true
	_, err = r.EvaluateOrder(o, h, compliance.Snapshot{})
	if !errors.Is(err, errCannotPlaceLeverageOrder) {
		t.Error(err)
	}

	r.MaximumLeverage = decimal.NewFromInt(33)
	r.CurrencySettings[e][a][p.Base.Item][p.Quote.Item].MaxLeverageRate = decimal.NewFromInt(33)
	_, err = r.EvaluateOrder(o, h, compliance.Snapshot{})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}

	r.MaximumLeverage = decimal.NewFromInt(33)
	r.CurrencySettings[e][a][p.Base.Item][p.Quote.Item].MaxLeverageRate = decimal.NewFromInt(33)

	_, err = r.EvaluateOrder(o, h, compliance.Snapshot{
		Orders: []compliance.SnapshotOrder{
			{
				Order: &gctorder.Detail{
					Leverage: 3,
				},
			},
		},
	})
	if !errors.Is(err, errCannotPlaceLeverageOrder) {
		t.Error(err)
	}

	h = append(h, holdings.Holding{Pair: p, BaseValue: decimal.NewFromInt(1337)}, holdings.Holding{Pair: p, BaseValue: decimal.NewFromFloat(1337.42)})
	r.CurrencySettings[e][a][p.Base.Item][p.Quote.Item].MaximumHoldingRatio = decimal.NewFromFloat(0.1)
	_, err = r.EvaluateOrder(o, h, compliance.Snapshot{})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}

	h = append(h, holdings.Holding{Pair: currency.NewPair(currency.DOGE, currency.LTC), BaseValue: decimal.NewFromInt(1337)})
	_, err = r.EvaluateOrder(o, h, compliance.Snapshot{})
	if !errors.Is(err, errCannotPlaceLeverageOrder) {
		t.Error(err)
	}
}
