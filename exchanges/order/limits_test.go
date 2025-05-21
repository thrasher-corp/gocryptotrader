package order

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

var (
	btcusd = currency.NewBTCUSD()
	ltcusd = currency.NewPair(currency.LTC, currency.USD)
	btcltc = currency.NewPair(currency.BTC, currency.LTC)
)

func TestLoadLimits(t *testing.T) {
	t.Parallel()
	e := ExecutionLimits{}
	err := e.LoadLimits(nil)
	assert.ErrorIs(t, err, errCannotLoadLimit)

	invalidAsset := []MinMaxLevel{
		{
			Pair:              btcusd,
			MinPrice:          100000,
			MaxPrice:          1000000,
			MinimumBaseAmount: 1,
			MaximumBaseAmount: 10,
		},
	}
	err = e.LoadLimits(invalidAsset)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	invalidPairLoading := []MinMaxLevel{
		{
			Asset:             asset.Spot,
			MinPrice:          100000,
			MaxPrice:          1000000,
			MinimumBaseAmount: 1,
			MaximumBaseAmount: 10,
		},
	}

	err = e.LoadLimits(invalidPairLoading)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	newLimits := []MinMaxLevel{
		{
			Pair:              btcusd,
			Asset:             asset.Spot,
			MinPrice:          100000,
			MaxPrice:          1000000,
			MinimumBaseAmount: 1,
			MaximumBaseAmount: 10,
		},
	}

	err = e.LoadLimits(newLimits)
	require.NoError(t, err)

	badLimit := []MinMaxLevel{
		{
			Pair:              btcusd,
			Asset:             asset.Spot,
			MinPrice:          2,
			MaxPrice:          1,
			MinimumBaseAmount: 1,
			MaximumBaseAmount: 10,
		},
	}

	err = e.LoadLimits(badLimit)
	require.ErrorIs(t, err, errInvalidPriceLevels)

	badLimit = []MinMaxLevel{
		{
			Pair:              btcusd,
			Asset:             asset.Spot,
			MinPrice:          1,
			MaxPrice:          2,
			MinimumBaseAmount: 10,
			MaximumBaseAmount: 9,
		},
	}

	err = e.LoadLimits(badLimit)
	require.ErrorIs(t, err, errInvalidAmountLevels)

	goodLimit := []MinMaxLevel{
		{
			Pair:  btcusd,
			Asset: asset.Spot,
		},
	}

	err = e.LoadLimits(goodLimit)
	require.NoError(t, err)

	noCompare := []MinMaxLevel{
		{
			Pair:              btcusd,
			Asset:             asset.Spot,
			MinimumBaseAmount: 10,
		},
	}

	err = e.LoadLimits(noCompare)
	require.NoError(t, err)

	noCompare = []MinMaxLevel{
		{
			Pair:     btcusd,
			Asset:    asset.Spot,
			MinPrice: 10,
		},
	}

	err = e.LoadLimits(noCompare)
	assert.NoError(t, err)
}

func TestGetOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	e := ExecutionLimits{}
	_, err := e.GetOrderExecutionLimits(asset.Spot, btcusd)
	require.ErrorIs(t, err, ErrExchangeLimitNotLoaded)

	newLimits := []MinMaxLevel{
		{
			Pair:              btcusd,
			Asset:             asset.Spot,
			MinPrice:          100000,
			MaxPrice:          1000000,
			MinimumBaseAmount: 1,
			MaximumBaseAmount: 10,
		},
	}

	err = e.LoadLimits(newLimits)
	require.NoError(t, err)

	_, err = e.GetOrderExecutionLimits(asset.Futures, ltcusd)
	require.ErrorIs(t, err, ErrCannotValidateAsset)

	_, err = e.GetOrderExecutionLimits(asset.Spot, ltcusd)
	require.ErrorIs(t, err, errExchangeLimitBase)

	_, err = e.GetOrderExecutionLimits(asset.Spot, btcltc)
	require.ErrorIs(t, err, errExchangeLimitQuote)

	tt, err := e.GetOrderExecutionLimits(asset.Spot, btcusd)
	require.NoError(t, err)
	assert.Equal(t, newLimits[0].MaximumBaseAmount, tt.MaximumBaseAmount)
	assert.Equal(t, newLimits[0].MinimumBaseAmount, tt.MinimumBaseAmount)
	assert.Equal(t, newLimits[0].MaxPrice, tt.MaxPrice)
	assert.Equal(t, newLimits[0].MinPrice, tt.MinPrice)
}

func TestCheckLimit(t *testing.T) {
	t.Parallel()
	e := ExecutionLimits{}
	err := e.CheckOrderExecutionLimits(asset.Spot, btcusd, 1337, 1337, Limit)
	require.NoError(t, err)

	newLimits := []MinMaxLevel{
		{
			Pair:              btcusd,
			Asset:             asset.Spot,
			MinPrice:          100000,
			MaxPrice:          1000000,
			MinimumBaseAmount: 1,
			MaximumBaseAmount: 10,
		},
	}

	err = e.LoadLimits(newLimits)
	require.NoError(t, err)

	err = e.CheckOrderExecutionLimits(asset.Futures, ltcusd, 1337, 1337, Limit)
	require.ErrorIs(t, err, ErrCannotValidateAsset)

	err = e.CheckOrderExecutionLimits(asset.Spot, ltcusd, 1337, 1337, Limit)
	require.ErrorIs(t, err, ErrCannotValidateBaseCurrency)

	err = e.CheckOrderExecutionLimits(asset.Spot, btcltc, 1337, 1337, Limit)
	require.ErrorIs(t, err, ErrCannotValidateQuoteCurrency)

	err = e.CheckOrderExecutionLimits(asset.Spot, btcusd, 1337, 9, Limit)
	require.ErrorIs(t, err, ErrPriceBelowMin)

	err = e.CheckOrderExecutionLimits(asset.Spot, btcusd, 1000001, 9, Limit)
	require.ErrorIs(t, err, ErrPriceExceedsMax)

	err = e.CheckOrderExecutionLimits(asset.Spot, btcusd, 999999, .5, Limit)
	require.ErrorIs(t, err, ErrAmountBelowMin)

	err = e.CheckOrderExecutionLimits(asset.Spot, btcusd, 999999, 11, Limit)
	require.ErrorIs(t, err, ErrAmountExceedsMax)

	err = e.CheckOrderExecutionLimits(asset.Spot, btcusd, 999999, 7, Limit)
	require.NoError(t, err)

	err = e.CheckOrderExecutionLimits(asset.Spot, btcusd, 999999, 7, Market)
	assert.NoError(t, err)
}

func TestConforms(t *testing.T) {
	t.Parallel()
	var tt MinMaxLevel
	err := tt.Conforms(0, 0, Limit)
	require.NoError(t, err)

	tt = MinMaxLevel{
		MinNotional: 100,
	}

	err = tt.Conforms(1, 1, Limit)
	require.ErrorIs(t, err, ErrNotionalValue)

	err = tt.Conforms(200, .5, Limit)
	require.NoError(t, err)

	tt.PriceStepIncrementSize = 0.001
	err = tt.Conforms(200.0001, .5, Limit)
	require.ErrorIs(t, err, ErrPriceExceedsStep)
	err = tt.Conforms(200.004, .5, Limit)
	require.NoError(t, err)

	tt.AmountStepIncrementSize = 0.001
	err = tt.Conforms(200, .0002, Limit)
	require.ErrorIs(t, err, ErrAmountExceedsStep)
	err = tt.Conforms(200000, .003, Limit)
	require.NoError(t, err)

	tt.MinimumBaseAmount = 1
	tt.MaximumBaseAmount = 10
	tt.MarketMinQty = 1.1
	tt.MarketMaxQty = 9.9

	err = tt.Conforms(200000, 1, Market)
	require.ErrorIs(t, err, ErrMarketAmountBelowMin)

	err = tt.Conforms(200000, 10, Market)
	require.ErrorIs(t, err, ErrMarketAmountExceedsMax)

	tt.MarketStepIncrementSize = 10
	err = tt.Conforms(200000, 9.1, Market)
	require.ErrorIs(t, err, ErrMarketAmountExceedsStep)
	tt.MarketStepIncrementSize = 1
	err = tt.Conforms(200000, 9.1, Market)
	assert.NoError(t, err)
}

func TestConformToDecimalAmount(t *testing.T) {
	t.Parallel()
	var tt MinMaxLevel
	require.True(t, tt.ConformToDecimalAmount(decimal.NewFromFloat(1.001)).Equal(decimal.NewFromFloat(1.001)))

	tt = MinMaxLevel{}
	val := tt.ConformToDecimalAmount(decimal.NewFromInt(1))
	assert.True(t, val.Equal(decimal.NewFromInt(1))) // If there is no step amount set, this should not change the inputted amount

	tt.AmountStepIncrementSize = 0.001
	val = tt.ConformToDecimalAmount(decimal.NewFromFloat(1.001))
	assert.True(t, val.Equal(decimal.NewFromFloat(1.001)))

	val = tt.ConformToDecimalAmount(decimal.NewFromFloat(0.0001))
	assert.True(t, val.IsZero())

	val = tt.ConformToDecimalAmount(decimal.NewFromFloat(0.7777))
	assert.True(t, val.Equal(decimal.NewFromFloat(0.777)))

	tt.AmountStepIncrementSize = 100
	val = tt.ConformToDecimalAmount(decimal.NewFromInt(100))
	assert.True(t, val.Equal(decimal.NewFromInt(100)))

	val = tt.ConformToDecimalAmount(decimal.NewFromInt(200))
	assert.True(t, val.Equal(decimal.NewFromInt(200)))
	val = tt.ConformToDecimalAmount(decimal.NewFromInt(150))
	assert.True(t, val.Equal(decimal.NewFromInt(100)))
}

func TestConformToAmount(t *testing.T) {
	t.Parallel()
	var tt MinMaxLevel
	require.Equal(t, 1.001, tt.ConformToAmount(1.001))

	tt = MinMaxLevel{}
	val := tt.ConformToAmount(1)
	assert.Equal(t, 1.0, val, "ConformToAmount should return the same value with no step amount set")

	tt.AmountStepIncrementSize = 0.001
	val = tt.ConformToAmount(1.001)
	assert.Equal(t, 1.001, val)

	val = tt.ConformToAmount(0.0001)
	assert.Zero(t, val)

	val = tt.ConformToAmount(0.7777)
	assert.Equal(t, 0.777, val)

	tt.AmountStepIncrementSize = 100
	val = tt.ConformToAmount(100)
	assert.Equal(t, 100.0, val)

	val = tt.ConformToAmount(200)
	require.Equal(t, 200.0, val)
	val = tt.ConformToAmount(150)
	assert.Equal(t, 100.0, val)
}
