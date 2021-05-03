package slippage

import (
	"testing"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/bitstamp"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func TestRandomSlippage(t *testing.T) {
	t.Parallel()
	resp := EstimateSlippagePercentage(80, 100)
	if resp < 0.8 || resp > 1 {
		t.Error("expected result > 0.8 and < 100")
	}
}

func TestCalculateSlippageByOrderbook(t *testing.T) {
	t.Parallel()
	b := bitstamp.Bitstamp{}
	b.SetDefaults()
	cp := currency.NewPair(currency.BTC, currency.USD)
	ob, err := b.FetchOrderbook(cp, asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	amountOfFunds := 1000.0
	feeRate := 0.03
	price, amount := CalculateSlippageByOrderbook(ob, gctorder.Buy, amountOfFunds, feeRate)
	if price*amount+(price*amount*feeRate) > amountOfFunds {
		t.Error("order size must be less than funds")
	}
}
