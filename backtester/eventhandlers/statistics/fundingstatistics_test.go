package statistics

import (
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

func TestCalculateFundingStatistics(t *testing.T) {
	t.Parallel()
	_, err := CalculateFundingStatistics(nil, nil, decimal.Zero, gctkline.OneHour)
	assert.ErrorIs(t, err, common.ErrNilPointer)

	f, err := funding.SetupFundingManager(&engine.ExchangeManager{}, true, true, false)
	assert.NoError(t, err)

	item, err := funding.CreateItem("binance", asset.Spot, currency.BTC, decimal.NewFromInt(1337), decimal.Zero)
	assert.NoError(t, err)

	err = f.AddItem(item)
	assert.NoError(t, err)

	item2, err := funding.CreateItem("binance", asset.Spot, currency.USD, decimal.NewFromInt(1337), decimal.Zero)
	assert.NoError(t, err)

	err = f.AddItem(item2)
	assert.NoError(t, err)

	_, err = CalculateFundingStatistics(f, nil, decimal.Zero, gctkline.OneHour)
	assert.ErrorIs(t, err, common.ErrNilPointer)

	usdKline := gctkline.Item{
		Exchange: "binance",
		Pair:     currency.NewBTCUSD(),
		Asset:    asset.Spot,
		Interval: gctkline.OneHour,
		Candles: []gctkline.Candle{
			{
				Time: time.Now().Add(-time.Hour),
			},
			{
				Time: time.Now(),
			},
		},
	}
	dfk := &kline.DataFromKline{
		Base: &data.Base{},
		Item: &usdKline,
	}
	err = dfk.Load()
	assert.NoError(t, err)

	err = f.AddUSDTrackingData(dfk)
	assert.ErrorIs(t, err, funding.ErrUSDTrackingDisabled)

	cs := make(map[key.ExchangeAssetPair]*CurrencyPairStatistic)
	_, err = CalculateFundingStatistics(f, cs, decimal.Zero, gctkline.OneHour)
	assert.NoError(t, err)

	f, err = funding.SetupFundingManager(&engine.ExchangeManager{}, true, false, false)
	assert.NoError(t, err)

	err = f.AddItem(item)
	assert.NoError(t, err)

	err = f.AddItem(item2)
	assert.NoError(t, err)

	err = f.AddUSDTrackingData(dfk)
	require.NoError(t, err, "AddUSDTrackingData must not error")

	cs[key.NewExchangeAssetPair("binance", asset.Spot, currency.NewPair(currency.LTC, currency.USD))] = &CurrencyPairStatistic{}
	_, err = CalculateFundingStatistics(f, cs, decimal.Zero, gctkline.OneHour)
	assert.ErrorIs(t, err, errMissingSnapshots)

	err = f.CreateSnapshot(usdKline.Candles[0].Time)
	assert.NoError(t, err)

	err = f.CreateSnapshot(usdKline.Candles[1].Time)
	require.NoError(t, err, "CreateSnapshot must not error")

	cs[key.NewExchangeAssetPair("binance", asset.Spot, currency.NewPair(currency.LTC, currency.USD))] = &CurrencyPairStatistic{}
	_, err = CalculateFundingStatistics(f, cs, decimal.Zero, gctkline.OneHour)
	assert.NoError(t, err)
}

func TestCalculateIndividualFundingStatistics(t *testing.T) {
	_, err := CalculateIndividualFundingStatistics(true, nil, nil)
	assert.ErrorIs(t, err, common.ErrNilPointer)

	_, err = CalculateIndividualFundingStatistics(true, &funding.ReportItem{}, nil)
	assert.NoError(t, err)

	_, err = CalculateIndividualFundingStatistics(false, &funding.ReportItem{}, nil)
	assert.ErrorIs(t, err, errMissingSnapshots)

	ri := &funding.ReportItem{
		Snapshots: []funding.ItemSnapshot{
			{
				USDValue: decimal.NewFromInt(1337),
			},
			{},
		},
	}
	rs := []relatedCurrencyPairStatistics{
		{
			isBaseCurrency: false,
			stat:           nil,
		},
		{
			isBaseCurrency: true,
			stat:           &CurrencyPairStatistic{},
		},
	}
	_, err = CalculateIndividualFundingStatistics(false, ri, rs)
	assert.ErrorIs(t, err, common.ErrNilPointer)

	rs[0].stat = &CurrencyPairStatistic{}
	ri.USDInitialFunds = decimal.NewFromInt(1000)
	ri.USDFinalFunds = decimal.NewFromInt(1337)
	_, err = CalculateIndividualFundingStatistics(false, ri, rs)
	assert.ErrorIs(t, err, errMissingSnapshots)

	cp := currency.NewBTCUSD()
	ri.USDPairCandle = &kline.DataFromKline{
		Base: &data.Base{},
		Item: &gctkline.Item{
			Exchange:       testExchange,
			Pair:           cp,
			UnderlyingPair: cp,
			Asset:          asset.Spot,
			Interval:       gctkline.OneHour,
			Candles: []gctkline.Candle{
				{
					Time: time.Now().Add(-time.Hour),
				},
				{
					Time: time.Now(),
				},
			},
			SourceJobID:     uuid.UUID{},
			ValidationJobID: uuid.UUID{},
		},
	}
	err = ri.USDPairCandle.Load()
	assert.NoError(t, err)

	_, err = CalculateIndividualFundingStatistics(false, ri, rs)
	assert.NoError(t, err)

	ri.Asset = asset.Futures
	_, err = CalculateIndividualFundingStatistics(false, ri, rs)
	assert.NoError(t, err)

	ri.IsCollateral = true
	_, err = CalculateIndividualFundingStatistics(false, ri, rs)
	assert.NoError(t, err)
}

func TestFundingStatisticsPrintResults(t *testing.T) {
	f := FundingStatistics{}
	err := f.PrintResults(false)
	assert.ErrorIs(t, err, common.ErrNilPointer)

	funds, err := funding.SetupFundingManager(&engine.ExchangeManager{}, true, true, false)
	assert.NoError(t, err)

	item1, err := funding.CreateItem("test", asset.Spot, currency.BTC, decimal.NewFromInt(1337), decimal.NewFromFloat(0.04))
	assert.NoError(t, err)

	item2, err := funding.CreateItem("test", asset.Spot, currency.LTC, decimal.NewFromInt(1337), decimal.NewFromFloat(0.04))
	assert.NoError(t, err)

	p, err := funding.CreatePair(item1, item2)
	assert.NoError(t, err)

	err = funds.AddPair(p)
	assert.NoError(t, err)

	f.Report, err = funds.GenerateReport()
	assert.NoError(t, err)

	err = f.PrintResults(false)
	assert.NoError(t, err)

	f.TotalUSDStatistics = &TotalFundingStatistics{}
	f.Report.DisableUSDTracking = false
	err = f.PrintResults(false)
	assert.ErrorIs(t, err, common.ErrNilPointer)

	f.TotalUSDStatistics = &TotalFundingStatistics{
		GeometricRatios:  &Ratios{},
		ArithmeticRatios: &Ratios{},
	}
	err = f.PrintResults(true)
	assert.NoError(t, err)
}
