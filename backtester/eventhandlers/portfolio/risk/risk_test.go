package risk

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/compliance"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func TestAssessHoldingsRatio(t *testing.T) {
	t.Parallel()
	ratio := assessHoldingsRatio(currency.NewBTCUSDT(), []holdings.Holding{
		{
			Pair:      currency.NewBTCUSDT(),
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

	ratio = assessHoldingsRatio(currency.NewBTCUSDT(), []holdings.Holding{
		{
			Pair:      currency.NewBTCUSDT(),
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
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	p := currency.NewBTCUSDT()
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
	r.CurrencySettings = make(map[key.ExchangeAssetPair]*CurrencySettings)
	_, err = r.EvaluateOrder(o, h, compliance.Snapshot{})
	assert.ErrorIs(t, err, errNoCurrencySettings)

	r.CurrencySettings[key.NewExchangeAssetPair(e, a, p)] = &CurrencySettings{
		MaximumOrdersWithLeverageRatio: decimal.NewFromFloat(0.3),
		MaxLeverageRate:                decimal.NewFromFloat(0.3),
		MaximumHoldingRatio:            decimal.NewFromFloat(0.3),
	}

	h = append(h, holdings.Holding{
		Pair:     p,
		BaseSize: decimal.NewFromInt(1),
	})
	_, err = r.EvaluateOrder(o, h, compliance.Snapshot{})
	assert.NoError(t, err)

	h = append(h, holdings.Holding{
		Pair: currency.NewPair(currency.DOGE, currency.USDT),
	})
	o.Leverage = decimal.NewFromFloat(1.1)
	r.CurrencySettings[key.NewExchangeAssetPair(e, a, p)].MaximumHoldingRatio = decimal.Zero
	_, err = r.EvaluateOrder(o, h, compliance.Snapshot{})
	assert.ErrorIs(t, err, errLeverageNotAllowed)

	r.CanUseLeverage = true
	_, err = r.EvaluateOrder(o, h, compliance.Snapshot{})
	assert.ErrorIs(t, err, errCannotPlaceLeverageOrder)

	r.MaximumLeverage = decimal.NewFromInt(33)
	r.CurrencySettings[key.NewExchangeAssetPair(e, a, p)].MaxLeverageRate = decimal.NewFromInt(33)
	_, err = r.EvaluateOrder(o, h, compliance.Snapshot{})
	assert.NoError(t, err)

	r.MaximumLeverage = decimal.NewFromInt(33)
	r.CurrencySettings[key.NewExchangeAssetPair(e, a, p)].MaxLeverageRate = decimal.NewFromInt(33)

	_, err = r.EvaluateOrder(o, h, compliance.Snapshot{
		Orders: []compliance.SnapshotOrder{
			{
				Order: &gctorder.Detail{
					Leverage: 3,
				},
			},
		},
	})
	assert.ErrorIs(t, err, errCannotPlaceLeverageOrder)

	h = append(h, holdings.Holding{Pair: p, BaseValue: decimal.NewFromInt(1337)}, holdings.Holding{Pair: p, BaseValue: decimal.NewFromFloat(1337.42)})
	r.CurrencySettings[key.NewExchangeAssetPair(e, a, p)].MaximumHoldingRatio = decimal.NewFromFloat(0.1)
	_, err = r.EvaluateOrder(o, h, compliance.Snapshot{})
	assert.NoError(t, err)

	h = append(h, holdings.Holding{Pair: currency.NewPair(currency.DOGE, currency.LTC), BaseValue: decimal.NewFromInt(1337)})
	_, err = r.EvaluateOrder(o, h, compliance.Snapshot{})
	assert.ErrorIs(t, err, errCannotPlaceLeverageOrder)
}
