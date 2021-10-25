package statistics

import (
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

func TestCalculateTotalUSDFundingStatistics(t *testing.T) {
	t.Parallel()
	_, err := CalculateFundingStatistics(nil, nil, decimal.Zero)
	if !errors.Is(err, common.ErrNilArguments) {
		t.Errorf("received %v expected %v", err, common.ErrNilArguments)
	}
	f := funding.SetupFundingManager(true, false)
	item, err := funding.CreateItem("binance", asset.Spot, currency.BTC, decimal.NewFromInt(1337), decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received %v expected %v", err, nil)
	}
	err = f.AddItem(item)
	if !errors.Is(err, nil) {
		t.Errorf("received %v expected %v", err, nil)
	}

	_, err = CalculateFundingStatistics(f, nil, decimal.Zero)
	if !errors.Is(err, common.ErrNilArguments) {
		t.Errorf("received %v expected %v", err, common.ErrNilArguments)
	}
}

func TestCalculateFundingStatistics(t *testing.T) {

}

func TestCalculateIndividualFundingStatistics(t *testing.T) {
	_, err := CalculateIndividualFundingStatistics(true, nil, nil)
	if !errors.Is(err, common.ErrNilArguments) {
		t.Errorf("received %v expected %v", err, common.ErrNilArguments)
	}

	_, err = CalculateIndividualFundingStatistics(true, &funding.ReportItem{}, nil)
	if !errors.Is(err, nil) {
		t.Errorf("received %v expected %v", err, nil)
	}

	_, err = CalculateIndividualFundingStatistics(false, &funding.ReportItem{}, nil)
	if !errors.Is(err, errMissingSnapshots) {
		t.Errorf("received %v expected %v", err, errMissingSnapshots)
	}

	ri := &funding.ReportItem{
		Snapshots: []funding.ItemSnapshot{
			{
				USDValue: decimal.NewFromInt(1337),
			},
			{
				USDValue: decimal.Zero,
			},
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
	if !errors.Is(err, common.ErrNilArguments) {
		t.Errorf("received %v expected %v", err, common.ErrNilArguments)
	}

	rs[0].stat = &CurrencyPairStatistic{}
	_, err = CalculateIndividualFundingStatistics(false, ri, rs)
	if !errors.Is(err, errMissingSnapshots) {
		t.Errorf("received %v expected %v", err, errMissingSnapshots)
	}

	ri.USDPairCandle = &kline.DataFromKline{
		Item: gctkline.Item{
			Interval: gctkline.OneHour,
			Candles: []gctkline.Candle{
				{
					Time: time.Now().Add(-time.Hour),
				},
				{
					Time: time.Now(),
				},
			},
		},
	}
	err = ri.USDPairCandle.Load()
	if !errors.Is(err, nil) {
		t.Errorf("received %v expected %v", err, nil)
	}
	_, err = CalculateIndividualFundingStatistics(false, ri, rs)
	if !errors.Is(err, nil) {
		t.Errorf("received %v expected %v", err, nil)
	}
}

func TestFundingStatisticsPrintResults(t *testing.T) {
	f := FundingStatistics{}
	err := f.PrintResults(false)
	if !errors.Is(err, common.ErrNilArguments) {
		t.Errorf("received %v expected %v", err, common.ErrNilArguments)
	}

	funds := funding.SetupFundingManager(true, true)
	item1, err := funding.CreateItem("test", asset.Spot, currency.BTC, decimal.NewFromInt(1337), decimal.NewFromFloat(0.04))
	if !errors.Is(err, nil) {
		t.Errorf("received %v expected %v", err, nil)
	}
	item2, err := funding.CreateItem("test", asset.Spot, currency.LTC, decimal.NewFromInt(1337), decimal.NewFromFloat(0.04))
	if !errors.Is(err, nil) {
		t.Errorf("received %v expected %v", err, nil)
	}
	p, err := funding.CreatePair(item1, item2)
	if !errors.Is(err, nil) {
		t.Errorf("received %v expected %v", err, nil)
	}
	err = funds.AddPair(p)
	if !errors.Is(err, nil) {
		t.Errorf("received %v expected %v", err, nil)
	}
	f.Report = funds.GenerateReport()
	err = f.PrintResults(false)
	if !errors.Is(err, nil) {
		t.Errorf("received %v expected %v", err, nil)
	}

	f.TotalUSDStatistics = &TotalFundingStatistics{}
	f.Report.DisableUSDTracking = false
	err = f.PrintResults(false)
	if !errors.Is(err, common.ErrNilArguments) {
		t.Errorf("received %v expected %v", err, common.ErrNilArguments)
	}

	f.TotalUSDStatistics = &TotalFundingStatistics{
		GeometricRatios:  &Ratios{},
		ArithmeticRatios: &Ratios{},
	}
	err = f.PrintResults(true)
	if !errors.Is(err, nil) {
		t.Errorf("received %v expected %v", err, nil)
	}
}
