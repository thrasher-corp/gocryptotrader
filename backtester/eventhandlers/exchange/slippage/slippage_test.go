package slippage

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/bitstamp"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func TestRandomSlippage(t *testing.T) {
	t.Parallel()
	resp := EstimateSlippagePercentage(decimal.NewFromInt(80), decimal.NewFromInt(100))
	if resp.LessThan(decimal.NewFromFloat(0.8)) || resp.GreaterThan(decimal.NewFromInt(1)) {
		t.Error("expected result > 0.8 and < 100")
	}
}

func TestCalculateSlippageByOrderbook(t *testing.T) {
	t.Parallel()
	b := bitstamp.Bitstamp{}
	b.SetDefaults()
	err := b.CurrencyPairs.SetAssetEnabled(asset.Spot, true)
	if err != nil {
		t.Fatal(err)
	}
	cp := currency.NewPair(currency.BTC, currency.USD)
	ob, err := b.FetchOrderbook(context.Background(), cp, asset.Spot)
	if err != nil {
		t.Error(err)
	}
	amountOfFunds := decimal.NewFromInt(1000)
	feeRate := decimal.NewFromFloat(0.03)
	price, amount, err := CalculateSlippageByOrderbook(ob, gctorder.Buy, amountOfFunds, feeRate)
	if err != nil {
		t.Fatal(err)
	}
	if price.Mul(amount).Add(price.Mul(amount).Mul(feeRate)).GreaterThan(amountOfFunds) {
		t.Error("order size must be less than funds")
	}
}
