package report

import (
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/statistics"
	evkline "github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func TestCreateUSDTotalsChart(t *testing.T) {
	t.Parallel()
	_, err := createUSDTotalsChart(nil, nil)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
	tt := time.Now()
	items := []statistics.ValueAtTime{
		{
			Time:  tt,
			Value: decimal.NewFromInt(1337),
			Set:   true,
		},
	}
	_, err = createUSDTotalsChart(items, nil)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
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
	if !errors.Is(err, nil) {
		t.Fatalf("received '%v' expected '%v'", err, nil)
	}
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
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
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
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	if !resp.ShowZeroDisclaimer {
		t.Error("expected ShowZeroDisclaimer")
	}
}

func TestCreatePNLCharts(t *testing.T) {
	t.Parallel()
	_, err := createPNLCharts(nil)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}

	tt := time.Now()
	var d Data
	d.Statistics = &statistics.Statistic{}
	d.Statistics.ExchangeAssetPairStatistics = make(map[string]map[asset.Item]map[*currency.Item]map[*currency.Item]*statistics.CurrencyPairStatistic)
	d.Statistics.ExchangeAssetPairStatistics[testExchange] = make(map[asset.Item]map[*currency.Item]map[*currency.Item]*statistics.CurrencyPairStatistic)
	d.Statistics.ExchangeAssetPairStatistics[testExchange][asset.Spot] = make(map[*currency.Item]map[*currency.Item]*statistics.CurrencyPairStatistic)
	d.Statistics.ExchangeAssetPairStatistics[testExchange][asset.Spot][currency.BTC.Item] = make(map[*currency.Item]*statistics.CurrencyPairStatistic)
	d.Statistics.ExchangeAssetPairStatistics[testExchange][asset.Spot][currency.BTC.Item][currency.USDT.Item] = &statistics.CurrencyPairStatistic{
		Events: []statistics.DataAtOffset{
			{
				PNL: &portfolio.PNLSummary{
					Result: gctorder.PNLResult{
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
		Pair:     currency.NewPair(currency.BTC, currency.USDT),
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
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	err = d.enhanceCandles()
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}

	_, err = createPNLCharts(d.Statistics.ExchangeAssetPairStatistics)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
}

func TestCreateFuturesSpotDiffChart(t *testing.T) {
	t.Parallel()
	_, err := createFuturesSpotDiffChart(nil)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}

	tt := time.Now()
	cp := currency.NewPair(currency.BTC, currency.USD)
	cp2 := currency.NewPair(currency.BTC, currency.DOGE)
	var d Data
	d.Statistics = &statistics.Statistic{}
	d.Statistics.ExchangeAssetPairStatistics = make(map[string]map[asset.Item]map[*currency.Item]map[*currency.Item]*statistics.CurrencyPairStatistic)
	d.Statistics.ExchangeAssetPairStatistics[testExchange] = make(map[asset.Item]map[*currency.Item]map[*currency.Item]*statistics.CurrencyPairStatistic)
	d.Statistics.ExchangeAssetPairStatistics[testExchange][asset.Spot] = make(map[*currency.Item]map[*currency.Item]*statistics.CurrencyPairStatistic)
	d.Statistics.ExchangeAssetPairStatistics[testExchange][asset.Spot][currency.BTC.Item] = make(map[*currency.Item]*statistics.CurrencyPairStatistic)
	d.Statistics.ExchangeAssetPairStatistics[testExchange][asset.Spot][currency.BTC.Item][currency.USD.Item] = &statistics.CurrencyPairStatistic{
		Currency: cp,
		Events: []statistics.DataAtOffset{
			{
				Time:      tt,
				DataEvent: &evkline.Kline{Close: decimal.NewFromInt(1337)},
				PNL: &portfolio.PNLSummary{
					Result: gctorder.PNLResult{
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
	d.Statistics.ExchangeAssetPairStatistics[testExchange][asset.Futures] = make(map[*currency.Item]map[*currency.Item]*statistics.CurrencyPairStatistic)
	d.Statistics.ExchangeAssetPairStatistics[testExchange][asset.Futures][currency.BTC.Item] = make(map[*currency.Item]*statistics.CurrencyPairStatistic)
	d.Statistics.ExchangeAssetPairStatistics[testExchange][asset.Futures][currency.BTC.Item][currency.DOGE.Item] = &statistics.CurrencyPairStatistic{
		UnderlyingPair: cp,
		Currency:       cp2,
		Events: []statistics.DataAtOffset{
			{
				Time:      tt,
				DataEvent: &evkline.Kline{Close: decimal.NewFromInt(1337)},
				PNL: &portfolio.PNLSummary{
					Result: gctorder.PNLResult{
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
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if len(charty.Data) == 0 {
		t.Error("expected data")
	}
}
