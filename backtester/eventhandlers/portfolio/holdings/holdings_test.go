package holdings

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
)

const testExchange = "binance"

func pair(t *testing.T) *funding.SpotPair {
	t.Helper()
	b, err := funding.CreateItem(testExchange, asset.Spot, currency.BTC, decimal.Zero, decimal.Zero)
	require.NoError(t, err, "funding.CreateItem must not error for base funding")
	q, err := funding.CreateItem(testExchange, asset.Spot, currency.USDT, decimal.NewFromInt(1337), decimal.Zero)
	require.NoError(t, err, "funding.CreateItem must not error for quote funding")
	p, err := funding.CreatePair(b, q)
	require.NoError(t, err, "funding.CreatePair must not error")
	return p
}

func collateral(t *testing.T) *funding.CollateralPair {
	t.Helper()
	b, err := funding.CreateItem(testExchange, asset.Spot, currency.BTC, decimal.Zero, decimal.Zero)
	require.NoError(t, err, "funding.CreateItem must not error for contract funding")
	q, err := funding.CreateItem(testExchange, asset.Spot, currency.USDT, decimal.NewFromInt(1337), decimal.Zero)
	require.NoError(t, err, "funding.CreateItem must not error for collateral funding")
	p, err := funding.CreateCollateral(b, q)
	require.NoError(t, err, "funding.CreateCollateral must not error")
	return p
}

func TestCreate(t *testing.T) {
	t.Parallel()
	_, err := Create(nil, pair(t))
	assert.ErrorIs(t, err, common.ErrNilEvent, "Create should error correctly for a nil event")

	_, err = Create(&fill.Fill{
		Base: &event.Base{AssetType: asset.Spot},
	}, pair(t))
	assert.NoError(t, err, "Create should not error for spot funding")

	_, err = Create(&fill.Fill{
		Base: &event.Base{AssetType: asset.Futures},
	}, collateral(t))
	assert.NoError(t, err, "Create should not error for futures funding")

	_, err = Create(&fill.Fill{
		Base: &event.Base{AssetType: asset.Spot},
	}, collateral(t))
	assert.ErrorIs(t, err, funding.ErrNotPair, "Create should error correctly when spot funding is collateral")

	_, err = Create(&fill.Fill{
		Base: &event.Base{AssetType: asset.Futures},
	}, pair(t))
	assert.ErrorIs(t, err, funding.ErrNotCollateral, "Create should error correctly when futures funding is a spot pair")

	_, err = Create(&fill.Fill{
		Base: &event.Base{AssetType: asset.Options},
	}, pair(t))
	assert.ErrorIs(t, err, asset.ErrNotSupported, "Create should error correctly for an unsupported asset type")
}

func TestUpdate(t *testing.T) {
	t.Parallel()
	h, err := Create(&fill.Fill{
		Base: &event.Base{AssetType: asset.Spot},
	}, pair(t))
	require.NoError(t, err, "Create must not error")

	t1 := h.Timestamp
	err = h.Update(&fill.Fill{
		Base: &event.Base{
			Time: time.Now(),
		},
	}, pair(t))
	assert.NoError(t, err, "Update should not error")
	assert.Falsef(t, t1.Equal(h.Timestamp), "Holding.Timestamp should be updated from %v to %v", t1, h.Timestamp)
}

func TestUpdateValue(t *testing.T) {
	t.Parallel()
	b := &event.Base{AssetType: asset.Spot}
	h, err := Create(&fill.Fill{
		Base: b,
	}, pair(t))
	require.NoError(t, err, "Create must not error")

	err = h.UpdateValue(nil)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer, "UpdateValue should error correctly for a nil event")

	h.BaseSize = decimal.NewFromInt(1)
	err = h.UpdateValue(&kline.Kline{
		Base:  b,
		Close: decimal.NewFromInt(1337),
	})
	assert.NoError(t, err, "UpdateValue should not error")
	assert.Truef(t, h.BaseValue.Equal(decimal.NewFromInt(1337)), "Holding.BaseValue should equal %v, actual %v", decimal.NewFromInt(1337), h.BaseValue)
}

func TestUpdateAssetTypes(t *testing.T) {
	t.Parallel()
	err := new(Holding).update(&fill.Fill{
		Base:  &event.Base{AssetType: asset.Spot},
		Order: new(order.Detail),
	}, collateral(t))
	assert.ErrorIs(t, err, funding.ErrNotPair, "Holding.update should error correctly when spot funding is collateral")

	err = new(Holding).update(&fill.Fill{
		Base:  &event.Base{AssetType: asset.Futures},
		Order: new(order.Detail),
	}, pair(t))
	assert.ErrorIs(t, err, funding.ErrNotCollateral, "Holding.update should error correctly when futures funding is a spot pair")

	err = new(Holding).update(&fill.Fill{
		Base:  &event.Base{AssetType: asset.Options},
		Order: new(order.Detail),
	}, pair(t))
	assert.ErrorIs(t, err, asset.ErrNotSupported, "Holding.update should error correctly for an unsupported asset type")

	contract, err := funding.CreateItem(testExchange, asset.Futures, currency.BTC, decimal.NewFromInt(4), decimal.Zero)
	require.NoError(t, err, "funding.CreateItem must not error for futures contract funding")
	margin, err := funding.CreateItem(testExchange, asset.Futures, currency.USDT, decimal.NewFromInt(1337), decimal.Zero)
	require.NoError(t, err, "funding.CreateItem must not error for futures collateral funding")
	funds, err := funding.CreateCollateral(contract, margin)
	require.NoError(t, err, "funding.CreateCollateral must not error for futures funding")
	h := new(Holding)
	err = h.update(&fill.Fill{
		Base:      &event.Base{AssetType: asset.Futures},
		Direction: order.Buy,
		Order: &order.Detail{
			Price:  2,
			Amount: 1,
			Fee:    3,
		},
	}, funds)
	assert.NoError(t, err, "Holding.update should not error for futures funding")
	assert.Truef(t, h.BaseSize.Equal(decimal.NewFromInt(4)), "Holding.BaseSize should equal %v, actual %v", decimal.NewFromInt(4), h.BaseSize)
	assert.Truef(t, h.QuoteSize.Equal(decimal.NewFromInt(1337)), "Holding.QuoteSize should equal %v, actual %v", decimal.NewFromInt(1337), h.QuoteSize)
	assert.Truef(t, h.BaseValue.Equal(decimal.NewFromInt(8)), "Holding.BaseValue should equal %v, actual %v", decimal.NewFromInt(8), h.BaseValue)
	assert.Truef(t, h.TotalFees.Equal(decimal.NewFromInt(3)), "Holding.TotalFees should equal %v, actual %v", decimal.NewFromInt(3), h.TotalFees)
	assert.True(t, h.BoughtAmount.IsZero(), "Holding.BoughtAmount should be zero for futures funding")
	assert.True(t, h.SoldAmount.IsZero(), "Holding.SoldAmount should be zero for futures funding")
}

func TestUpdateBuyStats(t *testing.T) {
	t.Parallel()
	b, err := funding.CreateItem(testExchange, asset.Spot, currency.BTC, decimal.NewFromInt(1), decimal.Zero)
	require.NoError(t, err, "funding.CreateItem must not error for base funding")
	q, err := funding.CreateItem(testExchange, asset.Spot, currency.USDT, decimal.NewFromInt(100), decimal.Zero)
	require.NoError(t, err, "funding.CreateItem must not error for quote funding")
	p, err := funding.CreatePair(b, q)
	require.NoError(t, err, "funding.CreatePair must not error")
	h, err := Create(&fill.Fill{
		Base: &event.Base{AssetType: asset.Spot},
	}, pair(t))
	require.NoError(t, err, "Create must not error")

	err = h.update(&fill.Fill{
		Base: &event.Base{
			Exchange:     testExchange,
			Time:         time.Now(),
			Interval:     gctkline.OneHour,
			CurrencyPair: currency.NewBTCUSDT(),
			AssetType:    asset.Spot,
		},
		Direction:           order.Buy,
		Amount:              decimal.NewFromInt(1),
		ClosePrice:          decimal.NewFromInt(500),
		VolumeAdjustedPrice: decimal.NewFromInt(500),
		PurchasePrice:       decimal.NewFromInt(500),
		Order: &order.Detail{
			Price:       500,
			Amount:      1,
			Exchange:    testExchange,
			OrderID:     "decimal.NewFromInt(1337)",
			Type:        order.Limit,
			Side:        order.Buy,
			Status:      order.New,
			AssetType:   asset.Spot,
			Date:        time.Now(),
			CloseTime:   time.Now(),
			LastUpdated: time.Now(),
			Pair:        currency.NewBTCUSDT(),
			Trades:      nil,
			Fee:         1,
		},
	}, p)
	require.NoError(t, err, "Holding.update must not error for an initial buy")
	assert.Truef(t, h.BaseSize.Equal(p.BaseAvailable()), "Holding.BaseSize should equal %v, actual %v", p.BaseAvailable(), h.BaseSize)
	assert.Truef(t, h.BaseValue.Equal(p.BaseAvailable().Mul(decimal.NewFromInt(500))), "Holding.BaseValue should equal %v, actual %v", p.BaseAvailable().Mul(decimal.NewFromInt(500)), h.BaseValue)
	assert.Truef(t, h.QuoteSize.Equal(decimal.NewFromInt(100)), "Holding.QuoteSize should equal %v, actual %v", decimal.NewFromInt(100), h.QuoteSize)
	assert.Truef(t, h.TotalValue.Equal(decimal.NewFromInt(600)), "Holding.TotalValue should equal %v, actual %v", decimal.NewFromInt(600), h.TotalValue)
	assert.Truef(t, h.BoughtAmount.Equal(decimal.NewFromInt(1)), "Holding.BoughtAmount should equal %v, actual %v", decimal.NewFromInt(1), h.BoughtAmount)
	assert.True(t, h.SoldAmount.IsZero(), "Holding.SoldAmount should be zero")
	assert.Truef(t, h.TotalFees.Equal(decimal.NewFromInt(1)), "Holding.TotalFees should equal %v, actual %v", decimal.NewFromInt(1), h.TotalFees)

	err = h.update(&fill.Fill{
		Base: &event.Base{
			Exchange:     testExchange,
			Time:         time.Now(),
			Interval:     gctkline.OneHour,
			CurrencyPair: currency.NewBTCUSDT(),
			AssetType:    asset.Spot,
		},
		Direction:           order.Buy,
		Amount:              decimal.MustFromFloat(0.5),
		ClosePrice:          decimal.NewFromInt(500),
		VolumeAdjustedPrice: decimal.NewFromInt(500),
		PurchasePrice:       decimal.NewFromInt(500),
		Order: &order.Detail{
			Price:       500,
			Amount:      0.5,
			Exchange:    testExchange,
			OrderID:     "decimal.NewFromInt(1337)",
			Type:        order.Limit,
			Side:        order.Buy,
			Status:      order.New,
			AssetType:   asset.Spot,
			Date:        time.Now(),
			CloseTime:   time.Now(),
			LastUpdated: time.Now(),
			Pair:        currency.NewBTCUSDT(),
			Trades:      nil,
			Fee:         0.5,
		},
	}, p)
	require.NoError(t, err, "Holding.update must not error for an additional buy")
	assert.Truef(t, h.BoughtAmount.Equal(decimal.MustFromFloat(1.5)), "Holding.BoughtAmount should equal %v, actual %v", decimal.MustFromFloat(1.5), h.BoughtAmount)
	assert.True(t, h.SoldAmount.IsZero(), "Holding.SoldAmount should be zero")
	assert.Truef(t, h.TotalFees.Equal(decimal.MustFromFloat(1.5)), "Holding.TotalFees should equal %v, actual %v", decimal.MustFromFloat(1.5), h.TotalFees)
}

func TestUpdateSellStats(t *testing.T) {
	t.Parallel()
	b, err := funding.CreateItem(testExchange, asset.Spot, currency.BTC, decimal.NewFromInt(1), decimal.Zero)
	require.NoError(t, err, "funding.CreateItem must not error for base funding")
	q, err := funding.CreateItem(testExchange, asset.Spot, currency.USDT, decimal.NewFromInt(100), decimal.Zero)
	require.NoError(t, err, "funding.CreateItem must not error for quote funding")
	p, err := funding.CreatePair(b, q)
	require.NoError(t, err, "funding.CreatePair must not error")

	h, err := Create(&fill.Fill{
		Base: &event.Base{AssetType: asset.Spot},
	}, p)
	require.NoError(t, err, "Create must not error")

	err = h.update(&fill.Fill{
		Base: &event.Base{
			Exchange:     testExchange,
			Time:         time.Now(),
			Interval:     gctkline.OneHour,
			CurrencyPair: currency.NewBTCUSDT(),
			AssetType:    asset.Spot,
		},
		Direction:           order.Buy,
		Amount:              decimal.NewFromInt(1),
		ClosePrice:          decimal.NewFromInt(500),
		VolumeAdjustedPrice: decimal.NewFromInt(500),
		PurchasePrice:       decimal.NewFromInt(500),
		Order: &order.Detail{
			Price:       500,
			Amount:      1,
			Exchange:    testExchange,
			OrderID:     "decimal.NewFromInt(1337)",
			Type:        order.Limit,
			Side:        order.Buy,
			Status:      order.New,
			AssetType:   asset.Spot,
			Date:        time.Now(),
			CloseTime:   time.Now(),
			LastUpdated: time.Now(),
			Pair:        currency.NewBTCUSDT(),
			Fee:         1,
		},
	}, p)
	require.NoError(t, err, "Holding.update must not error for a buy")
	assert.Truef(t, h.BaseSize.Equal(decimal.NewFromInt(1)), "Holding.BaseSize should equal %v, actual %v", decimal.NewFromInt(1), h.BaseSize)
	assert.Truef(t, h.BaseValue.Equal(decimal.NewFromInt(500)), "Holding.BaseValue should equal %v, actual %v", decimal.NewFromInt(500), h.BaseValue)
	assert.Truef(t, h.QuoteInitialFunds.Equal(decimal.NewFromInt(100)), "Holding.QuoteInitialFunds should equal %v, actual %v", decimal.NewFromInt(100), h.QuoteInitialFunds)
	assert.Truef(t, h.QuoteSize.Equal(decimal.NewFromInt(100)), "Holding.QuoteSize should equal %v, actual %v", decimal.NewFromInt(100), h.QuoteSize)
	assert.Truef(t, h.TotalValue.Equal(decimal.NewFromInt(600)), "Holding.TotalValue should equal %v, actual %v", decimal.NewFromInt(600), h.TotalValue)
	assert.Truef(t, h.BoughtAmount.Equal(decimal.NewFromInt(1)), "Holding.BoughtAmount should equal %v, actual %v", decimal.NewFromInt(1), h.BoughtAmount)
	assert.True(t, h.SoldAmount.IsZero(), "Holding.SoldAmount should be zero")
	assert.Truef(t, h.TotalFees.Equal(decimal.NewFromInt(1)), "Holding.TotalFees should equal %v, actual %v", decimal.NewFromInt(1), h.TotalFees)

	err = h.update(&fill.Fill{
		Base: &event.Base{
			Exchange:     testExchange,
			Time:         time.Now(),
			Interval:     gctkline.OneHour,
			CurrencyPair: currency.NewBTCUSDT(),
			AssetType:    asset.Spot,
		},
		Direction:           order.Sell,
		Amount:              decimal.NewFromInt(1),
		ClosePrice:          decimal.NewFromInt(500),
		VolumeAdjustedPrice: decimal.NewFromInt(500),
		PurchasePrice:       decimal.NewFromInt(500),
		Order: &order.Detail{
			Price:       500,
			Amount:      1,
			Exchange:    testExchange,
			OrderID:     "decimal.NewFromInt(1337)",
			Type:        order.Limit,
			Side:        order.Sell,
			Status:      order.New,
			AssetType:   asset.Spot,
			Date:        time.Now(),
			CloseTime:   time.Now(),
			LastUpdated: time.Now(),
			Pair:        currency.NewBTCUSDT(),
			Trades:      nil,
			Fee:         1,
		},
	}, p)
	require.NoError(t, err, "Holding.update must not error for a sell")
	assert.Truef(t, h.BoughtAmount.Equal(decimal.NewFromInt(1)), "Holding.BoughtAmount should equal %v, actual %v", decimal.NewFromInt(1), h.BoughtAmount)
	assert.Truef(t, h.SoldAmount.Equal(decimal.NewFromInt(1)), "Holding.SoldAmount should equal %v, actual %v", decimal.NewFromInt(1), h.SoldAmount)
	assert.Truef(t, h.TotalFees.Equal(decimal.NewFromInt(2)), "Holding.TotalFees should equal %v, actual %v", decimal.NewFromInt(2), h.TotalFees)
}
