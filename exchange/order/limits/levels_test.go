package limits

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func TestConforms(t *testing.T) {
	t.Parallel()
	tt := &MinMaxLevel{}
	err := tt.Validate(0, 0, order.Limit)
	require.NoError(t, err)

	tt = &MinMaxLevel{
		MinNotional: 100,
	}
	err = tt.Validate(1, 1, order.Limit)
	assert.ErrorIs(t, err, ErrNotionalValue)

	err = tt.Validate(200, .5, order.Limit)
	assert.NoError(t, err)

	tt.PriceStepIncrementSize = 0.001
	err = tt.Validate(200.0001, .5, order.Limit)
	assert.ErrorIs(t, err, ErrPriceExceedsStep)

	err = tt.Validate(200.004, .5, order.Limit)
	assert.NoError(t, err)

	tt.AmountStepIncrementSize = 0.001
	err = tt.Validate(200, .0002, order.Limit)
	assert.ErrorIs(t, err, ErrAmountExceedsStep)
	err = tt.Validate(200000, .003, order.Limit)
	assert.NoError(t, err)

	tt.MinimumBaseAmount = 1
	tt.MaximumBaseAmount = 10
	tt.MarketMinQty = 1.1
	tt.MarketMaxQty = 9.9

	err = tt.Validate(200000, 1, order.Market)
	assert.ErrorIs(t, err, ErrMarketAmountBelowMin)

	err = tt.Validate(200000, 10, order.Market)
	assert.ErrorIs(t, err, ErrMarketAmountExceedsMax)

	tt.MarketStepIncrementSize = 10
	err = tt.Validate(200000, 9.1, order.Market)
	assert.ErrorIs(t, err, ErrMarketAmountExceedsStep)

	tt.MarketStepIncrementSize = 1
	err = tt.Validate(200000, 9.1, order.Market)
	assert.NoError(t, err)

	tt = &MinMaxLevel{
		MinimumBaseAmount: 0.1,
	}
	err = tt.Validate(0, 0, order.Market)
	assert.ErrorIs(t, err, ErrAmountBelowMin)

	tt.MaximumBaseAmount = 0.5
	err = tt.Validate(0, 0.6, order.Market)
	assert.ErrorIs(t, err, ErrAmountExceedsMax)

	tt.AmountStepIncrementSize = 0.1
	err = tt.Validate(0, 0.1337, order.Market)
	assert.ErrorIs(t, err, ErrAmountExceedsStep)

	tt = nil
	err = tt.Validate(0, 0, order.Limit)
	assert.NoError(t, err)
}

func TestConformToDecimalAmount(t *testing.T) {
	t.Parallel()
	tt := &MinMaxLevel{}
	val := tt.FloorAmountToStepIncrementDecimal(decimal.NewFromFloat(1.001))
	assert.Equal(t, "1.001", val.String())

	tt = &MinMaxLevel{}
	val = tt.FloorAmountToStepIncrementDecimal(decimal.NewFromInt(1))
	assert.Equal(t, "1", val.String())

	tt.AmountStepIncrementSize = 0.001
	val = tt.FloorAmountToStepIncrementDecimal(decimal.NewFromFloat(1.001))
	assert.Equal(t, "1.001", val.String())

	val = tt.FloorAmountToStepIncrementDecimal(decimal.NewFromFloat(0.0001))
	assert.Equal(t, "0", val.String())

	val = tt.FloorAmountToStepIncrementDecimal(decimal.NewFromFloat(0.7777))
	assert.Equal(t, "0.777", val.String())

	tt.AmountStepIncrementSize = 100
	val = tt.FloorAmountToStepIncrementDecimal(decimal.NewFromInt(100))
	assert.Equal(t, "100", val.String())

	val = tt.FloorAmountToStepIncrementDecimal(decimal.NewFromInt(200))
	assert.Equal(t, "200", val.String())

	val = tt.FloorAmountToStepIncrementDecimal(decimal.NewFromInt(150))
	assert.Equal(t, "100", val.String())

	tt = nil
	val = tt.FloorAmountToStepIncrementDecimal(decimal.NewFromInt(150))
	assert.Equal(t, "150", val.String())
}

func TestConformToAmount(t *testing.T) {
	t.Parallel()
	tt := &MinMaxLevel{}
	require.Equal(t, 1.001, tt.FloorAmountToStepIncrement(1.001))

	tt = &MinMaxLevel{}
	val := tt.FloorAmountToStepIncrement(1.0)
	assert.Equal(t, 1.0, val)

	tt.AmountStepIncrementSize = 0.001
	val = tt.FloorAmountToStepIncrement(1.001)
	assert.Equal(t, 1.001, val)

	val = tt.FloorAmountToStepIncrement(0.0001)
	assert.Zero(t, val)

	val = tt.FloorAmountToStepIncrement(0.7777)
	assert.Equal(t, 0.777, val)

	tt.AmountStepIncrementSize = 100
	val = tt.FloorAmountToStepIncrement(100)
	assert.Equal(t, 100.0, val)

	val = tt.FloorAmountToStepIncrement(200)
	assert.Equal(t, 200.0, val)

	val = tt.FloorAmountToStepIncrement(150)
	assert.Equal(t, 100.0, val)

	tt = nil
	val = tt.FloorAmountToStepIncrement(150)
	assert.Equal(t, 150.0, val)
}

func TestConformToPrice(t *testing.T) {
	t.Parallel()
	tt := &MinMaxLevel{}
	resp := tt.FloorPriceToStepIncrement(1.0)
	assert.Equal(t, 1.0, resp)

	tt.PriceStepIncrementSize = 1

	resp = tt.FloorPriceToStepIncrement(1.5)
	assert.Equal(t, 1.0, resp)

	resp = tt.FloorPriceToStepIncrement(0.5)
	assert.Equal(t, 0.0, resp)

	tt = nil
	resp = tt.FloorPriceToStepIncrement(1.0)
	assert.Equal(t, 1.0, resp)
}
