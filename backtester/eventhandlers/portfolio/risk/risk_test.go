package risk

import (
	"testing"

	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/compliance"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

func TestAssessHoldingsRatio(t *testing.T) {
	ratio := assessHoldingsRatio(currency.NewPair(currency.BTC, currency.USDT), []holdings.Holding{
		{
			Pair:          currency.NewPair(currency.BTC, currency.USDT),
			PositionsSize: 2,
		},
		{
			Pair:          currency.NewPair(currency.LTC, currency.USDT),
			PositionsSize: 2,
		},
	})
	if ratio != 0.5 {
		t.Errorf("expected %v received %v", 0.5, ratio)
	}

	ratio = assessHoldingsRatio(currency.NewPair(currency.BTC, currency.USDT), []holdings.Holding{
		{
			Pair:          currency.NewPair(currency.BTC, currency.USDT),
			PositionsSize: 1,
		},
		{
			Pair:          currency.NewPair(currency.LTC, currency.USDT),
			PositionsSize: 2,
		},
		{
			Pair:          currency.NewPair(currency.DOGE, currency.USDT),
			PositionsSize: 1,
		},
	})
	if ratio != 0.25 {
		t.Errorf("expected %v received %v", 0.25, ratio)
	}
}

func TestEvaluateOrder(t *testing.T) {
	r := Risk{}
	_, err := r.EvaluateOrder(nil, nil, compliance.Snapshot{})
	if err != nil && err.Error() != "received nil argument(s)" {
		t.Error(err)
	}
	o := &order.Order{}
	h := []holdings.Holding{}
	_, err = r.EvaluateOrder(o, h, compliance.Snapshot{})
	if err != nil {
		t.Error(err)
	}
	p := currency.NewPair(currency.BTC, currency.USDT)
	e := "binance"
	a := asset.Spot
	o.Exchange = e
	o.AssetType = a
	o.CurrencyPair = p
	r.MaximumHoldingRatio = make(map[string]map[asset.Item]map[currency.Pair]float64)
	r.MaximumHoldingRatio[e] = make(map[asset.Item]map[currency.Pair]float64)
	r.MaximumHoldingRatio[e][a] = make(map[currency.Pair]float64)
	r.MaximumHoldingRatio[e][a][p] = 0.3

	r.MaxLeverageRatio = make(map[string]map[asset.Item]map[currency.Pair]float64)
	r.MaxLeverageRatio[e] = make(map[asset.Item]map[currency.Pair]float64)
	r.MaxLeverageRatio[e][a] = make(map[currency.Pair]float64)
	r.MaxLeverageRatio[e][a][p] = 0.3

	r.MaxLeverageRate = make(map[string]map[asset.Item]map[currency.Pair]float64)
	r.MaxLeverageRate[e] = make(map[asset.Item]map[currency.Pair]float64)
	r.MaxLeverageRate[e][a] = make(map[currency.Pair]float64)
	r.MaxLeverageRate[e][a][p] = 0.3

	h = append(h, holdings.Holding{
		Pair:          p,
		PositionsSize: 1,
	})
	_, err = r.EvaluateOrder(o, h, compliance.Snapshot{})
	if err != nil {
		t.Error(err)
	}

	h = append(h, holdings.Holding{
		Pair:          currency.NewPair(currency.DOGE, currency.USDT),
		PositionsSize: 0,
	})
	_, err = r.EvaluateOrder(o, h, compliance.Snapshot{})
	if err != nil && err.Error() != "proceeding with the order would put holdings ratio beyond its limit of 0.3 to 1 and cannot be placed" {
		t.Error(err)
	}

	o.Leverage = 1.1
	r.MaximumHoldingRatio[e][a][p] = 0
	_, err = r.EvaluateOrder(o, h, compliance.Snapshot{})
	if err != nil && err.Error() != "order says to use leverage, but it is not allowed" {
		t.Error(err)
	}
	r.CanUseLeverage = true
	_, err = r.EvaluateOrder(o, h, compliance.Snapshot{})
	if err != nil && err.Error() != "proceeding with the order would put leverage rate beyond its limit of 0.3 to 1.1 and cannot be placed" {
		t.Error(err)
	}

	r.MaxLeverageRate[e][a][p] = 1.2
	_, err = r.EvaluateOrder(o, h, compliance.Snapshot{})
	if err != nil && err.Error() != "proceeding with the order would put leverage rate beyond its limit of 0.3 to 1.1 and cannot be placed" {
		t.Error(err)
	}

	r.MaxLeverageRatio[e][a][p] = 1
	_, err = r.EvaluateOrder(o, h, compliance.Snapshot{})
	if err != nil && err.Error() != "proceeding with the order would put leverage rate beyond its limit of 0.3 to 1.1 and cannot be placed" {
		t.Error(err)
	}

}
