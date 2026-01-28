package limits

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

var happyKey = key.NewExchangeAssetPair("test", asset.Spot, currency.NewBTCUSDT())

func TestLoadLimits(t *testing.T) {
	t.Parallel()
	e := store{}
	err := e.load(nil)
	require.ErrorIs(t, err, ErrEmptyLevels)

	badKeyNoExchange := []MinMaxLevel{
		{
			Key:               key.NewExchangeAssetPair("", asset.Spot, currency.NewBTCUSDT()),
			MinPrice:          100000,
			MaxPrice:          1000000,
			MinimumBaseAmount: 1,
			MaximumBaseAmount: 10,
		},
	}
	err = e.load(badKeyNoExchange)
	assert.ErrorIs(t, err, errExchangeNameEmpty)

	badKeyNoAsset := []MinMaxLevel{
		{
			Key:               key.NewExchangeAssetPair("hi", 0, currency.NewBTCUSDT()),
			MinPrice:          100000,
			MaxPrice:          1000000,
			MinimumBaseAmount: 1,
			MaximumBaseAmount: 10,
		},
	}
	err = e.load(badKeyNoAsset)
	assert.ErrorIs(t, err, errAssetInvalid)

	badKeyNoPair := []MinMaxLevel{
		{
			Key:               key.NewExchangeAssetPair("hi", asset.Spot, currency.EMPTYPAIR),
			MinPrice:          100000,
			MaxPrice:          1000000,
			MinimumBaseAmount: 1,
			MaximumBaseAmount: 10,
		},
	}
	err = e.load(badKeyNoPair)
	assert.ErrorIs(t, err, errPairNotSet)

	happyLimit := []MinMaxLevel{
		{
			Key:               happyKey,
			MinPrice:          100000,
			MaxPrice:          1000000,
			MinimumBaseAmount: 1,
			MaximumBaseAmount: 10,
		},
	}

	err = e.load(happyLimit)
	assert.NoError(t, err)

	badLimit := []MinMaxLevel{
		{
			Key:               happyKey,
			MinPrice:          2,
			MaxPrice:          1,
			MinimumBaseAmount: 1,
			MaximumBaseAmount: 10,
		},
	}
	err = e.load(badLimit)
	assert.ErrorIs(t, err, errInvalidPriceLevels)

	badLimit = []MinMaxLevel{
		{
			Key:               happyKey,
			MinPrice:          1,
			MaxPrice:          2,
			MinimumBaseAmount: 10,
			MaximumBaseAmount: 9,
		},
	}
	err = e.load(badLimit)
	assert.ErrorIs(t, err, errInvalidAmountLevels)

	badLimit = []MinMaxLevel{
		{
			Key:                happyKey,
			MinimumQuoteAmount: 100,
			MaximumQuoteAmount: 10,
		},
	}
	err = e.load(badLimit)
	assert.ErrorIs(t, err, errInvalidQuoteLevels)
}

func TestGetOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	e := store{}
	_, err := e.getOrderExecutionLimits(happyKey)
	require.ErrorIs(t, err, ErrExchangeLimitNotLoaded)

	newLimits := []MinMaxLevel{
		{
			Key:               happyKey,
			MinPrice:          100000,
			MaxPrice:          1000000,
			MinimumBaseAmount: 1,
			MaximumBaseAmount: 10,
		},
	}
	err = e.load(newLimits)
	require.NoError(t, err)

	_, err = e.getOrderExecutionLimits(key.NewExchangeAssetPair("hi", asset.Futures, currency.NewBTCUSDT()))
	assert.ErrorIs(t, err, ErrOrderLimitNotFound)

	tt, err := e.getOrderExecutionLimits(happyKey)
	require.NoError(t, err)
	require.Equal(t, newLimits[0], tt)
}

func TestCheckLimit(t *testing.T) {
	t.Parallel()
	e := store{}
	err := e.checkOrderExecutionLimits(happyKey, 1337, 1337, order.Limit)
	require.ErrorIs(t, err, ErrExchangeLimitNotLoaded)

	newLimits := []MinMaxLevel{
		{
			Key:               happyKey,
			MinPrice:          100000,
			MaxPrice:          1000000,
			MinimumBaseAmount: 1,
			MaximumBaseAmount: 10,
		},
	}
	err = e.load(newLimits)
	assert.NoError(t, err)

	err = e.checkOrderExecutionLimits(key.NewExchangeAssetPair("test", asset.Futures, currency.NewBTCUSDT()), 1337, 1337, order.Limit)
	assert.ErrorIs(t, err, ErrOrderLimitNotFound)

	err = e.checkOrderExecutionLimits(happyKey, 1337, 9, order.Limit)
	assert.ErrorIs(t, err, ErrPriceBelowMin)

	err = e.checkOrderExecutionLimits(happyKey, 1000001, 9, order.Limit)
	assert.ErrorIs(t, err, ErrPriceExceedsMax)

	err = e.checkOrderExecutionLimits(happyKey, 999999, .5, order.Limit)
	assert.ErrorIs(t, err, ErrAmountBelowMin)

	err = e.checkOrderExecutionLimits(happyKey, 999999, 11, order.Limit)
	assert.ErrorIs(t, err, ErrAmountExceedsMax)

	err = e.checkOrderExecutionLimits(happyKey, 999999, 7, order.Limit)
	assert.NoError(t, err)

	err = e.checkOrderExecutionLimits(happyKey, 999999, 7, order.Market)
	assert.NoError(t, err)
}

// TestPublicLoadLimits does not run in parallel because it uses a global var
func TestPublicLoadLimits(t *testing.T) {
	err := Load(nil)
	assert.ErrorIs(t, err, ErrEmptyLevels)

	newLimits := []MinMaxLevel{
		{
			Key:               happyKey,
			MinPrice:          100000,
			MaxPrice:          1000000,
			MinimumBaseAmount: 1,
			MaximumBaseAmount: 10,
		},
	}
	err = Load(newLimits)
	require.NoError(t, err)
}

// TestPublicGetOrderExecutionLimits does not run in parallel because it uses a global var
func TestPublicGetOrderExecutionLimits(t *testing.T) {
	newLimits := []MinMaxLevel{
		{
			Key:               happyKey,
			MinPrice:          100000,
			MaxPrice:          1000000,
			MinimumBaseAmount: 1,
			MaximumBaseAmount: 10,
		},
	}
	err := Load(newLimits)
	require.NoError(t, err)

	resp, err := GetOrderExecutionLimits(happyKey)
	assert.NoError(t, err)
	assert.Equal(t, newLimits[0], resp)
}

// TestPublicCheckOrderExecutionLimits does not run in parallel because it uses a global var
func TestPublicCheckOrderExecutionLimits(t *testing.T) {
	newLimits := []MinMaxLevel{
		{
			Key:               happyKey,
			MinPrice:          100000,
			MaxPrice:          1000000,
			MinimumBaseAmount: 1,
			MaximumBaseAmount: 10,
		},
	}
	err := Load(newLimits)
	require.NoError(t, err)

	err = CheckOrderExecutionLimits(happyKey, 1, 1, order.Market)
	assert.NoError(t, err)
}
