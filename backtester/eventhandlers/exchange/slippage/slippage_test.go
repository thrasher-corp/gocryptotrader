package slippage

import (
	"testing"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/bitstamp"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func TestRandomSlippage(t *testing.T) {
	resp := EstimateSlippagePercentage(80, 100)
	if resp < 0.8 || resp > 1 {
		t.Error("expected result > 0.8 and < 100")
	}
}

func TestCalculateSlippageByOrderbook(t *testing.T) {
	b := bitstamp.Bitstamp{}
	b.SetDefaults()
	b.Verbose = false
	cp := currency.NewPair(currency.BTC, currency.USD)
	ob, err := b.FetchOrderbook(cp, asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	ticker, err := b.FetchTicker(cp, asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	rate := CalculateSlippageByOrderbook(ob, gctorder.Buy, ticker.High)
	if rate == 0 {
		t.Error("expected updated rate")
	}
	buyPrice := ticker.High * rate
	if buyPrice < ticker.High {
		t.Error("slipped price must be higher than original price")
	}

	rate = CalculateSlippageByOrderbook(ob, gctorder.Sell, ticker.Low)
	if rate == 0 {
		t.Error("expected updated rate")
	}
	sellPrice := ticker.Low * rate
	if sellPrice > ticker.Low {
		t.Error("slipped price must be lower than original price")
	}
}
