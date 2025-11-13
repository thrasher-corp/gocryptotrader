package report

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/statistics"
	evkline "github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func TestCreateUSDTotalsChart(t *testing.T) {
	t.Parallel()
	_, err := createUSDTotalsChart(nil, nil)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	tt := time.Now()
	items := []statistics.ValueAtTime{
		{
			Time:  tt,
			Value: decimal.NewFromInt(1337),
			Set:   true,
		},
	}
	_, err = createUSDTotalsChart(items, nil)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	stats := []statistics.FundingItemStatistics{
		{
			ReportItem: &funding.ReportItem{
				Snapshots: []funding.ItemSnapshot{
					{
						Time:     tt,
						USDValue: decimal.NewFromInt(1337),
					},
				},
			},
		},
	}
	resp, err := createUSDTotalsChart(items, stats)
	require.NoError(t, err)

	if len(resp.Data) == 0 {
		t.Fatal("expected not nil")
	}
	if resp.Data[0].Name != "Total USD value" {
		t.Error("expected not nil")
	}
	if resp.Data[0].LinePlots[0].Value != 1337 {
		t.Error("expected not nil")
	}
}

func TestCreateHoldingsOverTimeChart(t *testing.T) {
	t.Parallel()
	_, err := createHoldingsOverTimeChart(nil)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	tt := time.Now()
	items := []statistics.FundingItemStatistics{
		{
			ReportItem: &funding.ReportItem{
				Exchange: "hello",
				Asset:    asset.Spot,
				Currency: currency.BTC,
				Snapshots: []funding.ItemSnapshot{
					{
						Time:      tt,
						Available: decimal.NewFromInt(1337),
					},
					{
						Time: tt,
					},
				},
			},
		},
	}
	resp, err := createHoldingsOverTimeChart(items)
	assert.NoError(t, err)

	if !resp.ShowZeroDisclaimer {
		t.Error("expected ShowZeroDisclaimer")
	}
}

func TestCreatePNLCharts(t *testing.T) {
	t.Parallel()
	_, err := createPNLCharts(nil)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	tt := time.Now()
	var d Data
	d.Statistics = &statistics.Statistic{}
	d.Statistics.ExchangeAssetPairStatistics = make(map[key.ExchangeAssetPair]*statistics.CurrencyPairStatistic)
	d.Statistics.ExchangeAssetPairStatistics[key.NewExchangeAssetPair(testExchange, asset.Spot, currency.NewBTCUSDT())] = &statistics.CurrencyPairStatistic{
		Events: []statistics.DataAtOffset{
			{
				PNL: &portfolio.PNLSummary{
					Result: futures.PNLResult{
						Time:                  tt,
						UnrealisedPNL:         decimal.NewFromInt(1337),
						RealisedPNLBeforeFees: decimal.NewFromInt(1337),
						RealisedPNL:           decimal.NewFromInt(1337),
						Price:                 decimal.NewFromInt(1337),
						Exposure:              decimal.NewFromInt(1337),
						Direction:             gctorder.Short,
					},
				},
			},
		},
	}

	err = d.SetKlineData(&gctkline.Item{
		Exchange: testExchange,
		Pair:     currency.NewBTCUSDT(),
		Asset:    asset.Spot,
		Interval: gctkline.OneDay,
		Candles: []gctkline.Candle{
			{
				Time:   tt,
				Open:   1336,
				High:   1338,
				Low:    1336,
				Close:  1337,
				Volume: 1337,
			},
		},
	})
	assert.NoError(t, err)

	err = d.enhanceCandles()
	assert.NoError(t, err)

	_, err = createPNLCharts(d.Statistics.ExchangeAssetPairStatistics)
	assert.NoError(t, err)
}

func TestCreateFuturesSpotDiffChart(t *testing.T) {
	t.Parallel()
	_, err := createFuturesSpotDiffChart(nil)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	tt := time.Now()
	cp := currency.NewBTCUSD()
	cp2 := currency.NewPair(currency.BTC, currency.DOGE)
	var d Data
	d.Statistics = &statistics.Statistic{}
	d.Statistics.ExchangeAssetPairStatistics = make(map[key.ExchangeAssetPair]*statistics.CurrencyPairStatistic)
	d.Statistics.ExchangeAssetPairStatistics[key.NewExchangeAssetPair(testExchange, asset.Spot, currency.NewBTCUSD())] = &statistics.CurrencyPairStatistic{
		Currency: cp,
		Events: []statistics.DataAtOffset{
			{
				Time:      tt,
				DataEvent: &evkline.Kline{Close: decimal.NewFromInt(1337)},
				PNL: &portfolio.PNLSummary{
					Result: futures.PNLResult{
						Time:                  tt,
						UnrealisedPNL:         decimal.NewFromInt(1337),
						RealisedPNLBeforeFees: decimal.NewFromInt(1337),
						RealisedPNL:           decimal.NewFromInt(1337),
						Price:                 decimal.NewFromInt(1337),
						Exposure:              decimal.NewFromInt(1337),
						Direction:             gctorder.Buy,
					},
				},
			},
		},
	}
	d.Statistics.ExchangeAssetPairStatistics[key.NewExchangeAssetPair(testExchange, asset.Futures, currency.NewPair(currency.BTC, currency.DOGE))] = &statistics.CurrencyPairStatistic{
		UnderlyingPair: cp,
		Currency:       cp2,
		Events: []statistics.DataAtOffset{
			{
				Time:      tt,
				DataEvent: &evkline.Kline{Close: decimal.NewFromInt(1337)},
				PNL: &portfolio.PNLSummary{
					Result: futures.PNLResult{
						Time:                  tt,
						UnrealisedPNL:         decimal.NewFromInt(1337),
						RealisedPNLBeforeFees: decimal.NewFromInt(1337),
						RealisedPNL:           decimal.NewFromInt(1337),
						Price:                 decimal.NewFromInt(1337),
						Exposure:              decimal.NewFromInt(1337),
						Direction:             gctorder.Short,
					},
				},
			},
		},
	}

	charty, err := createFuturesSpotDiffChart(d.Statistics.ExchangeAssetPairStatistics)
	assert.NoError(t, err)

	if len(charty.Data) == 0 {
		t.Error("expected data")
	}
}
