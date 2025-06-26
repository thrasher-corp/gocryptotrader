package slippage

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/bitstamp"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func TestRandomSlippage(t *testing.T) {
	t.Parallel()
	resp := EstimateSlippagePercentage(decimal.NewFromInt(80), decimal.NewFromInt(100))
	assert.True(t, resp.GreaterThanOrEqual(decimal.NewFromFloat(0.8)), "result should be greater than or equal to 0.8")
	assert.True(t, resp.LessThan(decimal.NewFromInt(1)), "result should be less than 1")
}

func TestCalculateSlippageByOrderbook(t *testing.T) {
	t.Parallel()
	b := bitstamp.Exchange{}
	b.SetDefaults()

	cp := currency.NewBTCUSD()
	ob, err := b.UpdateOrderbook(t.Context(), cp, asset.Spot)
	require.NoError(t, err, "UpdateOrderbook must not error")

	amountOfFunds := decimal.NewFromInt(1000)
	feeRate := decimal.NewFromFloat(0.03)
	price, amount, err := CalculateSlippageByOrderbook(ob, gctorder.Buy, amountOfFunds, feeRate)
	require.NoError(t, err, "CalculateSlippageByOrderbook must not error")
	orderSize := price.Mul(amount).Add(price.Mul(amount).Mul(feeRate))
	assert.True(t, orderSize.LessThan(amountOfFunds), "order size should be less than funds")
}
