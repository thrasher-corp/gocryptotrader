package report

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/statistics"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

func TestCreateUSDTotalsChart(t *testing.T) {
	t.Parallel()
	if resp := createUSDTotalsChart(nil, nil); resp != nil {
		t.Error("expected nil")
	}
	tt := time.Now()
	items := []statistics.ValueAtTime{
		{
			Time:  tt,
			Value: decimal.NewFromInt(1337),
			Set:   true,
		},
	}
	if resp := createUSDTotalsChart(items, nil); resp != nil {
		t.Error("expected nil")
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
	resp := createUSDTotalsChart(items, stats)
	if resp == nil {
		t.Error("expected not nil")
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
	if resp := createHoldingsOverTimeChart(nil); resp != nil {
		t.Fatal("expected nil")
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
						Time:      tt,
						Available: decimal.Zero,
					},
				},
			},
		},
	}
	resp := createHoldingsOverTimeChart(items)
	if resp.AxisType != "linear" {
		t.Error("expected linear from zero available")
	}
}
